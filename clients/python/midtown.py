import requests
import json

"""
type Context map[string]string

type JobID string

type JobControl struct {
	MaxConcurrency         int
	StartTime              time.Time
	ContinueJobOnTaskError bool
	//RemoteCurDir            string
	WorkerNameRegex string
	//CompiledWorkerNameRegex *regexp.Regexp
	// TODO: consider OSRegex as well, to limit to Workers matching a particular OS/version
	//ProcessPriority int
	// TODO: later
	//AssignSingleTaskPerWorker bool
	//TaskWorkerAssignment      map[string][]uint32
	Priority int8          // higher value means higher priority
	Timeout  time.Duration // seconds
	//TaskTimeout          float64 // seconds
	//TaskSeemsHungTimeout uint32
	//AbandonedJobTimeout  uint32
	//MaxTaskReallocations uint8
}

type JobDefinition struct {
	Cmd         string
	Data        []interface{}
	Description string
	Ctx         *Context
	Ctrl        *JobControl
}
"""

#class JobDef(object):
	#def __init__(self, cmd):
	#	self.Cmd=cmd
		

class MidtownClient(object):
	def __init__(self):
		# TODO:
		pass
		
		
	def CreateJob(self, cmd, args, data, descript="", ctx=None, **ctrl):
		jobdef = dict(Cmd=cmd, Args=args, Data=data, Description=descript, Ctx=ctx, Ctrl=ctrl)
		#jobdef = dict(Cmd=cmd)
		#jobdef = JobDef(cmd)
		payload = json.dumps(jobdef)		
		print payload # TODO:
		r = requests.post("http://localhost:9997/jobs", data=payload, headers={"Content-Type":"application/json"})
		print(r.text)


def main():
	c = MidtownClient()
	c.CreateJob("python", '-c "import json,sys;sys.stdout.write(json.dumps(json.loads(sys.stdin.read())*2))"', range(10), "my job", {"path":"/foo/bar"}, 
	        Priority=1, MaxConcurrency=20)
	


def main2(): 
	r = requests.get("http://localhost:9997/workers")
	print r.status_code
	print r.json()
	print "-" * 40
	r = requests.get("http://localhost:9997/job/150107040846920585")
	print r.status_code
	print r.json()

if __name__ == "__main__":
	main() # TODO: