package backend

const walkTemplateString = `
<!DOCTYPE html>
<html>
	<head>
      <link id="favicon" rel="shortcut icon" type="image/png" href="data:image/png;base64,AAABAAEAEBAQAAEABAAoAQAAFgAAACgAAAAQAAAAIAAAAAEABAAAAAAAgAAAAAAAAAAAAAAAEAAAAAAAAACPj48Ax8fHAOPj4wA7OzsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAhERERERERACEREREREREAIRMRETMzEQAhExERERExACETERERETEAIRMRERERMQAhEzMREzMRACETERExEREAIRMRETEREQAhExERMRERACETMzMTMzEAIREREREREQAhERERERERACEREREREREAIiIiIiIiIiAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA">
      <title>{{.Path}}</title>
	</head>
	<body>
		<h1>Listing for dir: {{.Path}}</h1>
		<ul>
			<li><a href = "../">../</a></li>
			{{range .Entries}}
				<li>
				{{if .File}}
					<a href="/{{.DownloadPath}}">{{.Name}}</a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
				{{else}}
					<a href="{{.Name}}/">{{.Name}}/</a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<a href="/{{.DownloadPath}}">zip download</a>&nbsp;&nbsp;
				{{end}}
				<a href="/{{.GenTempLink}}">temp link</a></li>
				</li>
			{{end}}
		</ul>
	</body>
</html>`

const homeHTML = `<!doctype html><link id=favicon rel="shortcut icon" type=image/png href=data:image/png;base64,AAABAAEAEBAQAAEABAAoAQAAFgAAACgAAAAQAAAAIAAAAAEABAAAAAAAgAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAXl1cAP///wArKysAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMhEREREREREyAAAAAAAAATIAAAAAAAABMgAAAAAAAAEyACAAAiIgATIAIAAAACABMgAgAAAAIAEyACIiAiIgATIAIAACAAABMgAgAAIAAAEyACIiAiIgATIAAAAAAAABMgAAAAAAAAEyAAAAAAAAATIiIiIiIiIiMzMzMzMzMzMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA><style>body{width:9px;height:9px;position:absolute;top:0;bottom:0;left:0;right:0;margin:auto}</style><title>&#65279;</title><a href=/walk/>walk</a>`
