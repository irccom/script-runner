# script-runner script syntax
Quick rundown of the syntax you can use in a script-runner script.


### Comments
Lines that start with `#` are comments. There are some special types of comments that get used in generating the output HTML file though.

#### Script Title
The line that starts with `#~` is the title of the script, e.g.:

```
#~ Basic Connection script
```

#### Description Line
The line that starts with `#~d` is the description of the script, e.g.:

```
#~d Two clients connect and message each other
```


### Clients
Clients are referred to by their ID. For example, `c1`, `c2`, `dan`, `alice`, etc.

Clients are defined, and then actions can be taken using their ID.

#### Defining Clients
Lines that start with `!` define clients. For example:

```
! c1 c2 c3
```
This line defines the clients `c1`, `c2`, and `c3`.

```
! dan alice
```
This line defines the client `dan`, and the client `alice`.

```
!dan
```
This line is an error, there must be a space after the exclamation point.

#### Sending Traffic
Lines that start with a client's ID and a colon mean 'send this line for this client'. For example:

```
! dan
dan: CAP LS 302
```
The first line defines the `dan` client. The second line means "send the IRC message `CAP LS 302` to the server, from dan's connection".

#### Waiting For Responses
Lines that start with `->` wait for responses. After an action, you can wait for a response to the client that performed the action and/or other clients.

To wait for a response to the client that performed the action, simply use `->`. To wait for a response for a different client, also put the client's ID and a colon (as above) just after the arrow, e.g. `"-> dan:"`

After the arrow (and optional client identifier) comes a list of verbs to wait for. If any of these are seen, the script moves forward.

For example:

```
dan: PRIVMSG #test :Some message here
    -> privmsg
```
`dan` sends a `PRIVMSG` to the channel `#test`, and the line below waits to receive the message.

```
dan: MOTD
    -> 376 422
```
`dan` sends the `MOTD` command, and the line below waits to receive either the `RPL_ENDOFMOTD` `(376)` numeric, or the `ERR_NOMOTD` `(422)` numeric. After either is received, the script moves forward.

```
dan: PRIVMSG alice :This is a message!
    -> alice: privmsg
```
`dan` sends `alice` a `PRIVMSG`, and `alice` waits to receive the message.

### Disconnecting
Sometimes, you want a client to disconnect from the network (or want to wait for another client to do so). The `:<`, and `<?` lines let you do this. For example:

```
dan:< QUIT
```
`dan` sends the `QUIT` command, and the `<` on the sending line means that `dan` keeps processing incoming messages until they're disconnected from the network. Once the client has been disconnected, the script moves forward.

```
dan: KILL george
    <? george
```
`dan` sends the `KILL` command, and the line below means that `george` keeps processing incoming messages until they're disconnected from the network. Once `george` has been disconnected, the script moves forward.


### Future Extensions
By design, all the 'verbs' require a space after them to work correctly (for example, `!` requires a space after it, `->` does, etc). This is so that these verbs can be extended by adjusting these verbs. For example, to extend client definitions we might introduce a character after the `!` like `!=` or `!?` or something else.
