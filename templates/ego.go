package templates
import (
"fmt"
"io"
"github.com/kmanley/midtown/common"
)
//line activejobs.ego:1
 func ActiveJobs(w io.Writer, summList common.JobSummaryList) error  {
//line activejobs.ego:2
_, _ = fmt.Fprintf(w, "\n")
//line activejobs.ego:5
_, _ = fmt.Fprintf(w, "\n\n<html>\n  <body>\n    <h3>Active Jobs</h3>\n\t<table border=\"1\" cellpadding=\"0\" cellspacing=\"0\">\n\t\t<tr>\n\t\t\t<th>ID</th>\n\t\t\t<th>Description</th>\n\t\t\t<th>Created</th>\n\t\t\t<th>Started</th>\n\t\t\t<th>Finished</th>\n\t\t\t<th>#Tasks</th>\n\t\t\t<th>Pct</th>\n\t\t</tr>\n\t\t")
//line activejobs.ego:20

		for _, job := range summList {
		
//line activejobs.ego:23
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td><a href=\"/job/")
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "\">")
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "</a></td>\n\t\t\t<td>")
//line activejobs.ego:25
_, _ = fmt.Fprintf(w, "%v", job.Description)
//line activejobs.ego:25
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line activejobs.ego:26
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Created))
//line activejobs.ego:26
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line activejobs.ego:27
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Started))
//line activejobs.ego:27
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line activejobs.ego:28
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Finished))
//line activejobs.ego:28
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line activejobs.ego:29
_, _ = fmt.Fprintf(w, "%v", job.NumTasks)
//line activejobs.ego:29
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line activejobs.ego:30
_, _ = fmt.Fprintf(w, "%v", job.PctComplete)
//line activejobs.ego:30
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\n\t\t")
//line activejobs.ego:32
	
		}
		
//line activejobs.ego:35
_, _ = fmt.Fprintf(w, "\n\t</table>\n\n  </body>\n</html>")
return nil
}
//line completedjobs.ego:1
 func CompletedJobs(w io.Writer, dt string, summList common.JobSummaryList) error  {
//line completedjobs.ego:2
_, _ = fmt.Fprintf(w, "\n")
//line completedjobs.ego:5
_, _ = fmt.Fprintf(w, "\n\n<html>\n  <body>\n    <h3>Completed Jobs for ")
//line completedjobs.ego:8
_, _ = fmt.Fprintf(w, "%v", dt)
//line completedjobs.ego:8
_, _ = fmt.Fprintf(w, "</h3>\n\t<table border=\"1\" cellpadding=\"0\" cellspacing=\"0\">\n\t\t<tr>\n\t\t\t<th>ID</th>\n\t\t\t<th>Description</th>\n\t\t\t<th>Created</th>\n\t\t\t<th>Started</th>\n\t\t\t<th>Finished</th>\n\t\t\t<th>Duration</th>\n\t\t\t<th>#Tasks</th>\n\t\t\t<th>Pct</th>\n\t\t</tr>\n\t\t")
//line completedjobs.ego:21

		for _, job := range summList {
		
//line completedjobs.ego:24
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td><a href=\"/job/")
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "\">")
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "</a></td>\n\t\t\t<td>")
//line completedjobs.ego:26
_, _ = fmt.Fprintf(w, "%v", job.Description)
//line completedjobs.ego:26
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:27
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Created))
//line completedjobs.ego:27
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:28
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Started))
//line completedjobs.ego:28
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:29
_, _ = fmt.Fprintf(w, "%v", formatTime(job.Finished))
//line completedjobs.ego:29
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:30
_, _ = fmt.Fprintf(w, "%v", job.Runtime().String())
//line completedjobs.ego:30
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:31
_, _ = fmt.Fprintf(w, "%v", job.NumTasks)
//line completedjobs.ego:31
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line completedjobs.ego:32
_, _ = fmt.Fprintf(w, "%v", job.PctComplete)
//line completedjobs.ego:32
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\n\t\t")
//line completedjobs.ego:34
	
		}
		
//line completedjobs.ego:37
_, _ = fmt.Fprintf(w, "\n\t</table>\n\n  </body>\n</html>")
return nil
}
//line error.ego:1
 func Error(w io.Writer, err error) error  {
//line error.ego:2
_, _ = fmt.Fprintf(w, "\n\n<!DOCTYPE html>\n<html lang=\"en\">\n  <head>\n    <meta charset=\"utf-8\">\n    <title>midtown</title>\n  </head>\n\n  <body class=\"error\">\n    <div class=\"container\">\n      <div class=\"header\">\n        <h3 class=\"text-muted\">Error</h3>\n      </div>\n\n      An error has occurred: ")
//line error.ego:16
_, _ = fmt.Fprintf(w, "%v",  err )
//line error.ego:17
_, _ = fmt.Fprintf(w, "\n    </div> <!-- /container -->\n  </body>\n</html>")
return nil
}
//line job.ego:1
 func Job(w io.Writer, job *common.Job) error  {
//line job.ego:2
_, _ = fmt.Fprintf(w, "\n")
//line job.ego:5
_, _ = fmt.Fprintf(w, "\n\n<html>\n  <body>\n    <h3>Job ")
//line job.ego:8
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line job.ego:8
_, _ = fmt.Fprintf(w, "</h3>\n\t<table border=\"1\" cellpadding=\"0\" cellspacing=\"0\">\n\t\t<tr>\n\t\t\t<th>Attribute</th>\n\t\t\t<th>Value</th>\n\t\t</tr>\n\t\t<tr>\n\t\t\t<td>Description</td>\n\t\t\t<td>")
//line job.ego:16
_, _ = fmt.Fprintf(w, "%v", job.Description)
//line job.ego:16
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\n\t\t<tr>\n\t\t\t<td>Command</td>\n\t\t\t<td>")
//line job.ego:20
_, _ = fmt.Fprintf(w, "%v", job.Cmd)
//line job.ego:20
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\t\t\n\t\t<tr>\n\t\t\t<td>Args</td>\n\t\t\t<td>")
//line job.ego:24
_, _ = fmt.Fprintf(w, "%v", job.Args)
//line job.ego:24
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\t\t\n\t\t<tr>\n\t\t\t<td>Priority</td>\n\t\t\t<td>")
//line job.ego:28
_, _ = fmt.Fprintf(w, "%v", job.Ctrl.Priority)
//line job.ego:28
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\t\t\n\t</table>\n\t<h4>Tasks</h4>\n\t<table border=\"1\" cellpadding=\"0\" cellspacing=\"0\">\n\t\t<tr>\n\t\t\t<th>#</th>\n\t\t\t<th>Input</th>\n\t\t\t<th>Output</th>\n\t\t\t<th>Started</th>\n\t\t\t<th>Finished</th>\n\t\t\t<th>Elapsed</th>\n\t\t\t<th>Worker</th>\n\t\t\t<th>Error</th>\n\t\t\t<th>Stderr</th>\n\t\t</tr>\n\t\t")
//line job.ego:45

		for _, task := range job.Tasks {
		
//line job.ego:48
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td>")
//line job.ego:49
_, _ = fmt.Fprintf(w, "%v", task.Seq)
//line job.ego:49
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:50
_, _ = fmt.Fprintf(w, "%v", task.Indata)
//line job.ego:50
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:51
_, _ = fmt.Fprintf(w, "%v", task.Outdata)
//line job.ego:51
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:52
_, _ = fmt.Fprintf(w, "%v", formatTime(task.Started))
//line job.ego:52
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:53
_, _ = fmt.Fprintf(w, "%v", formatTime(task.Finished))
//line job.ego:53
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>TODO</td>\n\t\t\t<td>")
//line job.ego:55
_, _ = fmt.Fprintf(w, "%v", task.Worker)
//line job.ego:55
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:56
_, _ = fmt.Fprintf(w, "%v", task.Error)
//line job.ego:56
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line job.ego:57
_, _ = fmt.Fprintf(w, "%v", task.Stderr)
//line job.ego:57
_, _ = fmt.Fprintf(w, "</td><!--TODO: truncate if long-->\n\t\t</tr>\n\t\t")
//line job.ego:59
	
		}
		
//line job.ego:62
_, _ = fmt.Fprintf(w, "\n\t</table>\n\t\t\n  </body>\n</html>")
return nil
}
//line workers.ego:1
 func Workers(w io.Writer, workers common.WorkerList) error  {
//line workers.ego:2
_, _ = fmt.Fprintf(w, "\n")
//line workers.ego:6
_, _ = fmt.Fprintf(w, "\n\n<html>\n  <body>\n    <h3>Active Workers</h3>\n\t<table border=\"1\" cellpadding=\"0\" cellspacing=\"0\">\n\t\t<tr>\n\t\t\t<th>Name</th>\n\t\t\t<th>Curr Task</th>\n\t\t\t<th>OS</th>\n\t\t\t<th>Disk</th>\n\t\t\t<th>Mem</th>\n\t\t\t<th>#CPU</th>\n\t\t\t<th>Last Contact</th>\n\t\t</tr>\n\t\t")
//line workers.ego:21

		for _, worker := range workers {
		
//line workers.ego:24
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td>")
//line workers.ego:25
_, _ = fmt.Fprintf(w, "%v", worker.Name)
//line workers.ego:25
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t")
//line workers.ego:26
 if worker.CurrTask != nil { 
//line workers.ego:27
_, _ = fmt.Fprintf(w, "\n\t\t\t<td>")
//line workers.ego:27
_, _ = fmt.Fprintf(w, "%v", fmt.Sprintf("%s:%d", worker.CurrTask.Job, worker.CurrTask.Seq))
//line workers.ego:27
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t")
//line workers.ego:28
 } else { 
//line workers.ego:29
_, _ = fmt.Fprintf(w, "\n\t\t\t<td></td>\n\t\t\t")
//line workers.ego:30
}
//line workers.ego:31
_, _ = fmt.Fprintf(w, "\n\t\t\t<td>")
//line workers.ego:31
_, _ = fmt.Fprintf(w, "%v", worker.Stats.OSVersion)
//line workers.ego:31
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line workers.ego:32
_, _ = fmt.Fprintf(w, "%v", worker.Stats.CurrDisk)
//line workers.ego:32
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line workers.ego:33
_, _ = fmt.Fprintf(w, "%v", worker.Stats.CurrMem)
//line workers.ego:33
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line workers.ego:34
_, _ = fmt.Fprintf(w, "%v", worker.Stats.CurrCpu)
//line workers.ego:34
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
//line workers.ego:35
_, _ = fmt.Fprintf(w, "%v", formatTime(worker.LastContact))
//line workers.ego:35
_, _ = fmt.Fprintf(w, "</td>\n\t\t</tr>\n\t\t")
//line workers.ego:37
	
		}
		
//line workers.ego:40
_, _ = fmt.Fprintf(w, "\n\t</table>\n\n  </body>\n</html>")
return nil
}
