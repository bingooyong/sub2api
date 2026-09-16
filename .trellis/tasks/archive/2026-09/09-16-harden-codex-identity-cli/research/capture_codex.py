"""Capture only allowlisted metadata from an isolated CLI / local fake Responses server."""
import http.server
import json
import os
import platform
import signal
import subprocess
import tempfile
import threading
from pathlib import Path

observations = []
class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass
    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get('Content-Length', '0')))
        body = json.loads(raw)
        observations.append({
            'method': self.command, 'path': self.path, 'http_version': self.request_version,
            'headers': {k: self.headers[k] for k in ('user-agent', 'originator', 'version', 'content-type', 'accept') if k in self.headers},
            'header_names': sorted(k.lower() for k in self.headers),
            'body_keys': sorted(body),
            'input_item_types': sorted({x.get('type', 'message') for x in body.get('input', []) if isinstance(x, dict)}),
            'client_metadata_keys': sorted(body.get('client_metadata', {})),
            'stream': body.get('stream'), 'store': body.get('store'),
        })
        message = {'id':'msg_local','type':'message','status':'completed','role':'assistant','content':[{'type':'output_text','text':'OK','annotations':[]}]}
        response = {'id':'resp_local','object':'response','status':'completed','output':[message], 'usage':{'input_tokens':1,'output_tokens':1,'total_tokens':2}}
        events = [
            {'type':'response.created','response':dict(response, status='in_progress', output=[])},
            {'type':'response.output_item.added','output_index':0,'item':dict(message, status='in_progress',content=[])},
            {'type':'response.content_part.added','item_id':'msg_local','output_index':0,'content_index':0,'part':{'type':'output_text','text':'','annotations':[]}},
            {'type':'response.output_text.delta','item_id':'msg_local','output_index':0,'content_index':0,'delta':'OK'},
            {'type':'response.output_text.done','item_id':'msg_local','output_index':0,'content_index':0,'text':'OK'},
            {'type':'response.output_item.done','output_index':0,'item':message},
            {'type':'response.completed','response':response},
        ]
        wire = ''.join('event: '+e['type']+'\ndata: '+json.dumps(dict(e,sequence_number=i))+'\n\n' for i,e in enumerate(events)).encode()
        self.send_response(200)
        self.send_header('Content-Type','text/event-stream')
        self.send_header('Content-Length',str(len(wire)))
        self.end_headers()
        self.wfile.write(wire)

server = http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
threading.Thread(target=server.serve_forever,daemon=True).start()
cli_version = subprocess.check_output(['codex','--version'],text=True).strip()
with tempfile.TemporaryDirectory(prefix='codex-wire-check-') as scratch:
    env = os.environ.copy()
    env['CODEX_LOOPBACK_TEST_KEY'] = 'local-dummy-key'
    args = ['codex','exec','--ignore-user-config','--ephemeral','--skip-git-repo-check','--sandbox','read-only','-C',scratch,
            '-c','model_provider="loopback_test"','-c','model_providers.loopback_test.name="Local wire check"',
            '-c',f'model_providers.loopback_test.base_url="http://127.0.0.1:{server.server_port}/v1"',
            '-c','model_providers.loopback_test.wire_api="responses"',
            '-c','model_providers.loopback_test.env_key="CODEX_LOOPBACK_TEST_KEY"',
            '-c','model_providers.loopback_test.requires_openai_auth=false',
            '-m','gpt-5.5','Reply only OK. Do not use tools.']
    proc = subprocess.Popen(args,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,start_new_session=True)
    try:
        stdout, stderr = proc.communicate(timeout=30)
    except subprocess.TimeoutExpired:
        os.killpg(proc.pid, signal.SIGTERM)
        stdout, stderr = proc.communicate(timeout=5)
    result = subprocess.CompletedProcess(args, proc.returncode, stdout, stderr)
    # Diagnostic lines only, without request headers or prompt text.
    for line in stderr.splitlines():
        if any(word in line.lower() for word in ('error', 'reconnect', 'stream disconnected')):
            import sys
            print(line[:300], file=sys.stderr)

server.shutdown()
summary = {'cli_version':cli_version,'platform':platform.platform(),'mode':'exec','endpoint':'loopback HTTP with dummy key',
           'user_config_loaded':False,'returncode':result.returncode,'received_ok':result.stdout.strip()=='OK','requests':observations,
           'limitations':['Custom-provider HTTP observation only; not OAuth, production TLS, HTTP/2, or native WS verification.',
                          'No authorization values, prompt contents, session identifiers or installation identifiers retained.']}
print(json.dumps(summary,indent=2))
if result.returncode or not observations or result.stdout.strip()!='OK':
    raise SystemExit(1)
