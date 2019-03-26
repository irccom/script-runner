package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
	colorable "github.com/mattn/go-colorable"

	docopt "github.com/docopt/docopt-go"
	"github.com/irccom/test-framework/lib"
	"github.com/mgutz/ansi"
	"golang.org/x/text/cases"
)

// client colour, server colour
var ansiColorSchemes = [][]string{
	{"red+b", "red"},
	{"cyan+b", "cyan"},
	{"green+b", "green"},
	{"magenta+b", "magenta"},
	{"blue+b", "blue"},
	{"yellow+b", "yellow"},
}

func main() {
	usage := `testfw.

Usage:
	testfw run [options] <address> <script-filename>
	testfw print <script-filename>
	testfw -h | --help
	testfw --version

Options:
	--tls               Connect using TLS.
	--tls-noverify      Don't verify the provided TLS certificates.
	--no-colours        Disable coloured output.
	-h --help           Show this screen.
	--version           Show version.`

	arguments, _ := docopt.ParseDoc(usage)

	scriptFilename := arguments["<script-filename>"].(string)

	if arguments["print"].(bool) {
		// read script
		scriptBytes, err := ioutil.ReadFile(scriptFilename) // just pass the file name
		if err != nil {
			log.Fatal(err)
		}
		scriptString := string(scriptBytes)

		script, err := ReadScript(scriptString)
		if err != nil {
			log.Fatal(err)
		}

		// print script
		fmt.Println(script.String())
	}

	if arguments["run"].(bool) {
		address := arguments["<address>"].(string)
		useColours := !arguments["--no-colours"].(bool)

		// read script
		scriptBytes, err := ioutil.ReadFile(scriptFilename) // just pass the file name
		if err != nil {
			log.Fatal(err)
		}
		scriptString := string(scriptBytes)

		script, err := ReadScript(scriptString)
		if err != nil {
			log.Fatal(err)
		}

		// assign output colours to clients
		clientColours := map[string][]string{}
		colourableStdout := colorable.NewColorableStdout()

		if useColours {
			// ensure colours are applied consistently
			var clientIDsSorted []string
			for id := range script.Clients {
				clientIDsSorted = append(clientIDsSorted, id)
			}
			sort.Strings(clientIDsSorted)

			var colSchemeI int
			for _, id := range clientIDsSorted {
				clientColours[id] = ansiColorSchemes[colSchemeI]
				colSchemeI++
				if len(ansiColorSchemes) <= colSchemeI {
					colSchemeI = 0
				}
			}
		}

		// get additional connection config
		useTLS := arguments["--tls"].(bool)
		var tlsConfig *tls.Config
		if arguments["--tls-noverify"].(bool) {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		// make clients and connect 'em to the server
		sockets := make(map[string]*lib.Socket)
		for id := range script.Clients {
			socket, err := lib.ConnectSocket(address, useTLS, tlsConfig)
			if err != nil {
				log.Fatal("Could not connect client:", err.Error())
			}
			sockets[id] = socket
		}

		// run through actions
		for actionI, action := range script.Actions {
			socket := sockets[action.Client]

			// send line
			if action.LineToSend != "" {
				socket.SendLine(action.LineToSend)
				line := fmt.Sprintf("%s  -> %s", action.Client, action.LineToSend)
				if useColours {
					line = ansi.Color(line, clientColours[action.Client][0])
					fmt.Fprintln(colourableStdout, line)
				} else {
					fmt.Println(line)
				}
			}

			// wait for response
			if 0 < len(action.WaitAfterFor) {
				for {
					lineString, err := socket.GetLine()
					if err != nil {
						log.Fatal(fmt.Sprintf("Could not get line from server on action %d (%s):", actionI, action.Client), err.Error())
					}

					line, err := ircmsg.ParseLine(lineString)
					if err != nil {
						log.Fatal(fmt.Sprintf("Got malformed line from server on action %d (%s): [%s]", actionI, action.Client, lineString), err.Error())
					}

					verb := strings.ToLower(line.Command)

					// auto-respond to pings... in a dodgy, hacky way :<
					if verb == "ping" {
						socket.SendLine(fmt.Sprintf("PONG :%s", line.Params[0]))
						continue
					}

					out := fmt.Sprintf("%s <-  %s", action.Client, lineString)
					if useColours {
						out = ansi.Color(out, clientColours[action.Client][1])
						fmt.Fprintln(colourableStdout, out)
					} else {
						fmt.Println(out)
					}

					// found an action we're waiting for
					if action.WaitAfterFor[verb] {
						break
					}
				}
			}
		}

		// disconnect
		for _, socket := range sockets {
			socket.SendLine("QUIT")
			socket.Disconnect()
		}
	}
}

type ScriptAction struct {
	// client this action applies to
	Client string
	// line that this client should send
	LineToSend string
	// list of messages to wait for after sending the given line (if one is given)
	WaitAfterFor map[string]bool
}

type Script struct {
	Clients map[string]bool
	Actions []ScriptAction
}

func (s *Script) String() string {
	var t string

	// client list
	var clientIDs []string
	for id := range s.Clients {
		clientIDs = append(clientIDs, id)
	}
	sort.Strings(clientIDs)

	t += fmt.Sprintf("Clients: %s\n", strings.Join(clientIDs, ", "))

	// actions
	for _, action := range s.Actions {
		if action.LineToSend != "" {
			t += fmt.Sprintf("%s will send: %s\n", action.Client, action.LineToSend)
		}
		if 0 < len(action.WaitAfterFor) {
			var verbs []string
			for verb := range action.WaitAfterFor {
				verbs = append(verbs, verb)
			}
			sort.Strings(verbs)
			t += fmt.Sprintf("  %s will wait for: %s\n", action.Client, strings.Join(verbs, " "))
		}
	}

	return t
}

func NewScriptAction() ScriptAction {
	var sa ScriptAction
	sa.WaitAfterFor = make(map[string]bool)
	return sa
}

func ReadScript(t string) (*Script, error) {
	var s Script
	s.Clients = make(map[string]bool)

	lineNumber := 0
	for _, line := range strings.Split(t, "\n") {
		lineNumber++

		// remove junk at start of lines, we don't care about indentation
		line = strings.TrimLeft(line, " \t")

		// skip empty lines
		if len(line) < 1 {
			continue
		}

		// skip comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		// handle client definitions
		if strings.HasPrefix(line, "! ") {
			line = strings.TrimPrefix(line, "! ")

			ids := strings.Fields(line)
			if len(ids) < 1 {
				return nil, fmt.Errorf("No client IDs defined with [!] on line %d", lineNumber)
			}

			for _, id := range ids {
				id = cases.Fold().String(id)

				exists := s.Clients[id]
				if exists {
					return nil, fmt.Errorf("Client ID [%s] is redefined on line %d", id, lineNumber)
				}

				if strings.HasPrefix(id, "!") || strings.HasPrefix(id, "#") || strings.HasPrefix(id, "-") {
					return nil, fmt.Errorf("Client ID [%s] starts with a disallowed character", id)
				}

				if strings.ContainsAny(id, ":") {
					return nil, fmt.Errorf("Client ID [%s] contains a disallowed character", id)
				}

				// for safety, because we fold it
				if strings.ContainsAny(id, " \t") {
					return nil, fmt.Errorf("Client ID [%s] cannot contain whitespace", id)
				}

				s.Clients[id] = true
				// fmt.Println("New client", id, "defined")
			}

			continue
		}

		if strings.HasPrefix(line, "!") {
			return nil, fmt.Errorf("Malformed client definition line on line %d, must start with '! ' (including the space) [%s]", lineNumber, line)
		}

		// handle sync lines
		if strings.HasPrefix(line, "-> ") {
			if len(s.Actions) < 1 {
				return nil, fmt.Errorf("Sync line on line %d has no actions to sync against", lineNumber)
			}

			originalLine := line
			line = strings.TrimSpace(strings.TrimPrefix(line, "-> "))

			var clientID string

			if strings.Contains(line, ":") {
				foldedLine := cases.Fold().String(line)
				for id := range s.Clients {
					if strings.HasPrefix(foldedLine, id+":") {
						clientID = id
						break
					}
				}

				splitLine := strings.SplitN(line, ":", 2)
				line = splitLine[1]
			} else {
				lastAction := s.Actions[len(s.Actions)-1]
				// tl;dr you can't do this:
				//
				// c1 JOIN #channel
				//     -> c2: 141
				//     -> 134
				if lastAction.LineToSend != "" {
					clientID = lastAction.Client
				}
			}
			if clientID == "" {
				return nil, fmt.Errorf("Could not find matching client for sync line %d: [%s]", lineNumber, originalLine)
			}

			verbs := strings.Fields(line)
			newAction := NewScriptAction()
			newAction.Client = clientID
			for _, verb := range verbs {
				verb = strings.ToLower(verb)
				newAction.WaitAfterFor[verb] = true
			}

			s.Actions = append(s.Actions, newAction)
			continue
		}

		if strings.HasPrefix(line, "-") {
			return nil, fmt.Errorf("Malformed sync line on line %d, must start with '-> ' [%s]", lineNumber, line)
		}

		// handle action lines
		var actionLine bool
		for id := range s.Clients {
			if strings.HasPrefix(line, id+" ") {
				splitLine := strings.SplitN(line, " ", 2)
				newAction := NewScriptAction()
				newAction.Client = id
				newAction.LineToSend = splitLine[1]
				s.Actions = append(s.Actions, newAction)
				actionLine = true

			} else if strings.HasPrefix(line, id+"\t") {
				splitLine := strings.SplitN(line, "\t", 2)
				newAction := NewScriptAction()
				newAction.Client = id
				newAction.LineToSend = splitLine[1]
				s.Actions = append(s.Actions, newAction)
				actionLine = true
			}
		}
		if actionLine {
			continue
		}

		// fail on all other lines lol
		return nil, fmt.Errorf("Could not understand line %d: [%s]", lineNumber, line)
	}

	if len(s.Clients) == 0 {
		return nil, fmt.Errorf("No clients defined in the script")
	}

	return &s, nil
}
