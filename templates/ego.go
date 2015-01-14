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
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td>")
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line activejobs.ego:24
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
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
_, _ = fmt.Fprintf(w, "\n\t\t<tr>\n\t\t\t<td>")
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "%v", job.Id)
//line completedjobs.ego:25
_, _ = fmt.Fprintf(w, "</td>\n\t\t\t<td>")
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
