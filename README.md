# IRC Foundation IRCd Testing Framework
This is a framework that helps us perform tests using our [test servers](https://github.com/irccom/test-servers). More than anything, it's intended to simplify sending a consistent burst of traffic to a bunch of different servers, so humans can see the differences in the results and/or collect examples to populate the [IRC Foundation's Developer Docs](https://github.com/irccom/devdocs).

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

Between each line, a `PING/PONG` may be used to confirm timings (along with the `->` lines) and are automagically responded to and ignored. Because of this, this framework can't be used to test the `PING` or `PONG` commands/messages.

The tool outputs the traffic received by each client, delineated nicely.
