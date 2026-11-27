#!/usr/bin/env python3
# Bench-only (arm "verify"): the tester's writing call becomes three: the tester writes, a second reader (Opus, read-only,
# no hindsight) looks for a cheap wrong change each test would pass on, and, when it objects, the tester answers again once.
# Called by wrap/claude with the real claude's arguments; stdin is the task; stdout is the stream the engine reads.
import json,os,subprocess,sys,time
REAL=os.environ.get('WL_REAL_CLAUDE','claude')  # the real claude, never this wrapper
H=os.path.dirname(os.path.abspath(__file__))
args=sys.argv[1:]; task=sys.stdin.read()
keep=os.environ.get('WL_KEEP'); d=None
if keep:
    d=f"{keep}/{time.time_ns()}-{os.getpid()}"; os.makedirs(d,exist_ok=True)
    open(d+'/args','w').write('\n'.join(args)); open(d+'/cwd','w').write(os.getcwd()); open(d+'/stdin','w').write(task)
def save(name,text):
    if d: open(f'{d}/{name}','w').write(text)
def result(stream):
    for line in reversed(stream.splitlines()):
        try:
            m=json.loads(line)
            if m.get('type')=='result': return m.get('result') or '', m
        except Exception: pass
    return '', {}
p1=subprocess.run([REAL]+args,input=task,capture_output=True,text=True)
save('stdout',p1.stdout); ans,_=result(p1.stdout)
# an answer asked again by the engine (its feedback is in the task) is not read a second time
again=('send it again' in task) or ('Your answer was read, and refused' in task)
if p1.returncode!=0 or not ans.strip() or again or '- patch' not in ans and '- covers' not in ans:
    sys.stdout.write(p1.stdout); sys.exit(p1.returncode)
issue=task.split('\n## The code the')[0]
vp=subprocess.run([REAL,'-p','--model','opus','--effort','high','--output-format','json','--no-session-persistence','--tools','Read,Glob,Grep',
                   '--permission-mode','dontAsk','--strict-mcp-config','--setting-sources','user','--max-turns','25'],
                  input=open(H+'/prompt.txt').read()+'\n\n# The issue\n\n'+issue+"\n\n# The tester's answer\n\n"+ans,capture_output=True,text=True)
save('verifier.json',vp.stdout)
try: obj=json.loads(vp.stdout).get('result','')
except Exception: obj=''
if '- scenario:' not in obj:
    sys.stdout.write(p1.stdout); sys.exit(0)
task2=(task+"\n\n## Your first answer\n\n"+ans+"\n\n## A second reader's objections\n\n"+obj+
       "\n\nEach objection names a change to the code that would leave your test green without doing what its scenario says. "
       "For each: tighten the test, its setup or its assertion, so that it fails on that change; or keep it, and say in a `note` why the objection does not hold. "
       "Change nothing else. Answer again whole, the list alone: every file whole, every `covers` entry.\n")
save('stdin2',task2)
p2=subprocess.run([REAL]+args,input=task2,capture_output=True,text=True)
save('stdout2',p2.stdout); ans2,_=result(p2.stdout)
if p2.returncode==0 and ans2.strip(): sys.stdout.write(p2.stdout); sys.exit(0)
sys.stdout.write(p1.stdout); sys.exit(0)
