// 2019 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license
package lib

import "fmt"

const (
	htmlTemplate = `<html>
<head>
<title>%[1]s - IRC Foundation Test Framework</title>
<style>

:root {
	--sans-font-family: Trebuchet MS, Lucida Grande, Lucida Sans Unicode, Lucida Sans, Tahoma, sans-serif;
	--mono-font-family: monaco, Consolas, Lucida Console, monospace;

	--side-indent: 0.65em;
}

html {
	margin: 0;
	padding: 0;
}
body {
	font-family: var(--sans-font-family);
	display: flex;
	flex-direction: column;
	height: 100%%;
	margin: 0;
	padding: 0;
}
header {
	flex: 0 0 content;
	padding: 0 var(--side-indent);
	display: flex;
	flex-direction: column;
}
.content {
	flex: 1 1 auto;
	background: #aeae65;
	height: 100%%;
	width: 100%%;
}
footer {
	flex: 0 0 content;
	display: flex;
	padding: 0.3em var(--side-indent);
}


a {
	color: #217de4;
	font-style: italic;
	text-decoration: none;
}
h1 {
	font-size: 3em;
	color: #243847;
	padding: 0.5em 0;
	margin: 0;
}
.desc {
	display: block;
    color: #455e6e;
    margin-top: -0.8em;
    font-size: 1.05em;
    padding-bottom: 1.7em;
}

</style>
</head>
<body>

<header>
	<h1>%[1]s</h1>
	<span class="desc">%[2]s</span>
</header>

<div class="content">
	<div class="tab-bar">
		<div class="tabs">
		</div>
		<div class="options">
		</div>
	</div>
	<div class="tab-content">

	</div>
</div>

<footer>
	<a href="https://github.com/irccom/test-framework">IRC Foundation's test-framework</a>
</footer>

</body>
</html>`
)

// HTMLFromResults takes a set of results and outputs an HTML representation of those results.
func HTMLFromResults(script *Script, serverConfigs map[string]ServerConfig, scriptResults map[string]*ScriptResults) string {
	return fmt.Sprintf(htmlTemplate, script.Name, script.ShortDescription)
}
