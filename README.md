# IRC Foundation IRCd Script Runner
This is a framework that runs scripts over IRC servers (particularly our [test servers](https://github.com/irccom/test-servers)). It's an easy way to send a consistent burst of traffic to a bunch of different servers, and the output a file which can be easily looked through to see differences and collect examples to populate the [IRC Foundation's Developer Docs](https://github.com/irccom/devdocs).

It uses a very simple script format to send traffic.

Specifically:

    # this defines the client IDs in use
    ! <client_id>{ <client_id>}

    # this is a comment
    <client_id> <line to send>
        -> <message or numerics to wait for>
        -> <client_id>: <message or numerics the given client should wait for>

For example:

    ! c1 c2
    c1 NICK dan
    c1 USER d d d d
        -> 376 422
    c2 NICK alice
    c2 USER a a a a
        -> 376 422
    c1 PRIVMSG alice :Here is a line!
        -> c2: privmsg

Once connection registeration has completed, clients send `PING` messages to ensure that command responses are tracked correctly. Before registration has completed, the `PING` command cannot be used, so clients use the `->` lines. Because of this, the `->` lines are only be used to wait for things pre-registration (such as `CAP` response lines and the `-> 376 422` above) or after registration to ensure that other clients receive responses from one client's actions (the `-> c2: privmsg` in the above example).

Because of how we track commands and messages, this framework can't be used to test the `PING` command or `PONG` response.

The tool outputs the traffic received by each client, delineated nicely.


## Assumptions
This tool assumes:

- No other clients are connected to the server.
- Any spam-protection, filtering, or reconnection throttling has been disabled.

If these assumptions are not true, the bundled scripts may fail and break in surprising ways.
