// 2019 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license
package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/pkg/browser"

	"github.com/goshuirc/irc-go/ircmsg"
	colorable "github.com/mattn/go-colorable"

	docopt "github.com/docopt/docopt-go"
	"github.com/irccom/test-framework/lib"
	"github.com/mgutz/ansi"
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
	testfw run-multi [options] <settings-filename> <script-filename>
	testfw print <script-filename>
	testfw -h | --help
	testfw --version

Options:
	--tls               Connect using TLS.
	--tls-noverify      Don't verify the provided TLS certificates.
	--no-colours        Disable coloured output.
	--browser           Open the result HTML in the browser.
	--debug             Output extra debug lines.
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

		script, err := lib.ReadScript(scriptString)
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

		script, err := lib.ReadScript(scriptString)
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

	if arguments["run-multi"].(bool) {
		// read script
		scriptBytes, err := ioutil.ReadFile(scriptFilename) // just pass the file name
		if err != nil {
			log.Fatal(err)
		}
		scriptString := string(scriptBytes)

		script, err := lib.ReadScript(scriptString)
		if err != nil {
			log.Fatal(err)
		}

		// load config
		config, err := lib.LoadConfigFromFile(arguments["<settings-filename>"].(string))
		if err != nil {
			log.Fatalf("Could not read config: %s", err.Error())
		}

		// test out each server in order
		var serverIDsSorted []string
		for id := range config.Servers {
			serverIDsSorted = append(serverIDsSorted, id)
		}
		sort.Strings(serverIDsSorted)

		// make ScriptResults store
		scriptResults := make(map[string]*lib.ScriptResults)

		// print debug lines to work out exact output issues
		debug := arguments["--debug"].(bool)

		for _, id := range serverIDsSorted {
			info := config.Servers[id]
			fmt.Print("- ", info.DisplayName, " ...")

			// make script results
			sr := lib.NewScriptResults()

			// get additional connection config
			var tlsConfig *tls.Config
			if info.TLSSkipVerify {
				tlsConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
			}

			// make clients and connect 'em to the server
			if debug {
				fmt.Print("\n")
			}
			sockets := make(map[string]*lib.Socket)
			for id := range script.Clients {
				socket, err := lib.ConnectSocket(info.Address, info.UseTLS, tlsConfig)
				if err != nil {
					log.Fatal("Could not connect client:", err.Error())
				}
				sockets[id] = socket
				if debug {
					fmt.Println("Connected client", id)
				}
			}

			// registered tracks so we can switch to ping tracking (much more accurate)
			registered := make(map[string]bool)

			// used to let clients properly wait for other clients to receive responses
			var lastClientSent string

			// run through actions
			for actionI, action := range script.Actions {
				socket := sockets[action.Client]

				// send line
				if action.LineToSend == "" {
					srl := lib.ScriptResultLine{
						Type:    lib.ResultActionSync,
						Client:  action.Client,
						RawLine: "",
					}
					sr.Lines = append(sr.Lines, srl)
				} else {
					if debug {
						fmt.Println(action.Client, action.LineToSend)
					}
					socket.SendLine(action.LineToSend)
					srl := lib.ScriptResultLine{
						Type:    lib.ResultActionSync,
						Client:  action.Client,
						RawLine: action.LineToSend,
					}
					sr.Lines = append(sr.Lines, srl)
					if debug {
						fmt.Println(" -> sending")
					}
					lastClientSent = action.Client
				}

				// wait for response in old way
				if 0 < len(action.WaitAfterFor) && (!registered[action.Client] || lastClientSent != action.Client) {
					if debug {
						fmt.Println(" -", action.Client, "waiting in old way")
					}
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

						// mark registered
						if verb == "001" {
							registered[action.Client] = true
						}

						srl := lib.ScriptResultLine{
							Type:    lib.ResultIRCMessage,
							Client:  action.Client,
							RawLine: lineString,
						}
						sr.Lines = append(sr.Lines, srl)
						if debug {
							fmt.Println("  -", action.Client, "in:", verb, "   ", lineString)
						}

						// found an action we're waiting for
						if action.WaitAfterFor[verb] {
							if debug {
								fmt.Println("  -", action.Client, "in break")
							}
							break
						}
					}
				}

				// wait for response in new way once registered
				syncPingString := fmt.Sprintf("sync%d", actionI)
				if registered[action.Client] {
					socket.Send(nil, "", "PING", syncPingString)

					if debug {
						fmt.Println(" -", action.Client, "waiting in new way")
					}

					for {
						lineString, err := socket.GetLine()
						if err != nil {
							log.Fatal(fmt.Sprintf("Could not get new line from server on action %d (%s):", actionI, action.Client), err.Error())
						}

						line, err := ircmsg.ParseLine(lineString)
						if err != nil {
							log.Fatal(fmt.Sprintf("Got malformed new line from server on action %d (%s): [%s]", actionI, action.Client, lineString), err.Error())
						}

						verb := strings.ToLower(line.Command)

						// if response
						if verb == "pong" && line.Params[1] == syncPingString {
							break
						}

						// auto-respond to pings... in a dodgy, hacky way :<
						if verb == "ping" {
							socket.SendLine(fmt.Sprintf("PONG :%s", line.Params[0]))
							continue
						}

						srl := lib.ScriptResultLine{
							Type:    lib.ResultIRCMessage,
							Client:  action.Client,
							RawLine: lineString,
						}
						sr.Lines = append(sr.Lines, srl)
						if debug {
							fmt.Println("  -", action.Client, "in:", verb, "   ", lineString)
						}
					}
				}
			}

			// disconnect
			for _, socket := range sockets {
				socket.SendLine("QUIT")
				socket.Disconnect()
			}

			// store results
			scriptResults[id] = sr

			// print done line
			fmt.Println("OK!")
		}

		// create result file
		output := lib.HTMLFromResults(script, config.Servers, scriptResults)

		// output all results as a HTML file
		tmpfile, err := ioutil.TempFile("", "irc-test-framework.*.html")
		if err != nil {
			log.Fatal(err)
		}

		tmpfile.WriteString(output)
		tmpfile.Close()

		fmt.Println("\nResults are in:", tmpfile.Name())

		if arguments["--browser"].(bool) {
			browser.OpenFile(tmpfile.Name())
		}
	}
}
