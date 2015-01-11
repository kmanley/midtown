import requests
import json
import time
import logging
log = logging.getLogger(__name__)

# TODO: deal with marshaling time.Duration (e.g. job/task timeouts)
# TODO: deal with marshaling time.Date (e.g. StartAfter time)


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
		
		
class Error(Exception):
	pass

class Job(object):
	def __init__(self, client, jobid):
		self.client = client
		self.jobid = jobid 
		
	def GetJobResult(self):
		r = requests.get("http://%s:%d/result/%s" % (self.client.hostname, self.client.port, self.jobid))
		print r.status_code
		if r.status_code == 102:
			# job still working
			log.debug("job %s still working" % self.jobid)
			return None
		ret = json.loads(r.text)
		if len(ret) == 1:
			err = ret["Error"]
			log.error("job %s resulted in error: %s" % (self.jobid, err))
			raise Error(err)
		else:
			log.debug("successful job result: %s" % repr(ret))
			return ret
	
	# TODO: improve
	def Wait(self):
		while True:
			ret = self.GetJobResult()
			if ret:
				return ret
			time.sleep(1.0) # TODO:

class MidtownClient(object):
	def __init__(self, hostname, port):
		self.hostname = hostname
		self.port = port
		
	def CreateJob(self, cmd, args, data, descript="", ctx=None, **ctrl):
		ctx = ctx or {}
		jobdef = dict(Cmd=cmd, Args=args, Data=data, Description=descript, Ctx=ctx, Ctrl=ctrl)
		payload = json.dumps(jobdef)		
		log.debug("CreateJob payload: %s" % payload)
		r = requests.post("http://%s:%d/jobs" % (self.hostname, self.port), 
						data=payload, headers={"Content-Type":"application/json"})
		ret = json.loads(r.text)
		if type(ret) == dict:
			raise Error(ret["Error"])
		else:
			return Job(self, ret)

def main():
	c = MidtownClient("localhost", 9997)
	#c.CreateJob("python", ['-c', '"import json,sys;sys.stdout.write(json.dumps(json.loads(sys.stdin.read())*2));sys.stdout.flush()"'], 
	#					range(3), "my job", {"path":"/foo/bar"}, 
	#        Priority=1, MaxConcurrency=20)
	#c.CreateJob("echo", ["123"], range(3), "test")
	
	#job = c.CreateJob("python", ["-c", "import json,sys;job,seq,data,ctx=json.load(sys.stdin);json.dump(data*2,sys.stdout)"],
	#             range(20), Ctx={"foo":"bar"})
	job = c.CreateJob("python", ["-c", "import json,sys;job,seq,data,ctx=json.load(sys.stdin);import time;time.sleep(1);json.dump(data*2,sys.stdout)"],
	             range(20), Ctx={"foo":"bar"})				
	ret = job.Wait()
	print ret
	

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