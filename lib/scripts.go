// 2019 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license
package lib

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
)

// ScriptResults is the output from running a Script on a server.
type ScriptResults struct {
	Clients map[string]bool
	Lines   []ScriptResultLine
}

// NewScriptResults returns a new ScriptResults
func NewScriptResults() *ScriptResults {
	var sr ScriptResults
	sr.Clients = make(map[string]bool)
	return &sr
}

// ScriptResultLineType is a type of result line from running a script.
type ScriptResultLineType int

const (
	// ResultIRCMessage is a regular IRC message.
	ResultIRCMessage ScriptResultLineType = iota
	// ResultDisconnected is a client being disconnected from the server.
	ResultDisconnected = iota
	// ResultActionSync is a note that an 'action' has just been processed, used to sync
	//  results between servers.
	ResultActionSync = iota
)

// ScriptResultLine is a return IRC line or other action from the server.
type ScriptResultLine struct {
	// required
	Type   ScriptResultLineType
	Client string

	// optional
	RawLine string
}

// Script is a series of actions to perform on a server.
type Script struct {
	// metadata
	Name             string
	ShortDescription string

	// actual data
	Clients map[string]bool
	Actions []ScriptAction
}

// String is a text representation of the script's actions.
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

// ReadScript creates a Script from a given script file.
func ReadScript(t string) (*Script, error) {
	var s Script
	s.Clients = make(map[string]bool)

	lineNumber := 0
	for _, line := range strings.Split(t, "\n") {
		lineNumber++

		// remove junk at start of lines, we don't care about indentation
		line = strings.TrimLeft(line, " \t")

		// skip empty lines
		if len(strings.TrimSpace(line)) < 1 {
			continue
		}

		// get metadata lines
		if strings.HasPrefix(line, "#~ ") {
			line = strings.TrimPrefix(line, "#~ ")
			s.Name = strings.TrimSpace(line)
			continue
		}
		if strings.HasPrefix(line, "#~d ") {
			line = strings.TrimPrefix(line, "#~d ")
			s.ShortDescription = strings.TrimSpace(line)
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

// ScriptAction is an action to perform on the server, with the given client.
type ScriptAction struct {
	// client this action applies to
	Client string
	// line that this client should send
	LineToSend string
	// list of messages to wait for after sending the given line (if one is given)
	WaitAfterFor map[string]bool
}

// NewScriptAction returns a new ScriptAction.
func NewScriptAction() ScriptAction {
	var sa ScriptAction
	sa.WaitAfterFor = make(map[string]bool)
	return sa
}
