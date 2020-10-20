Fairly straight forward here,

The qualifier translator takes a list of qualifiers
and then parses them to generate a command. The command
is fairly straight forward. The resource_type is used
to pass the command on to its specific function and each
of the 'xCommand' functions return _another_ function
that you can pass a uri into and get a command from.
This is so that we handle the case of multiple URIs.

Fetching fetcher (name tbd) simply uses that to convert
and creates an action from it. Not quite wired up yet
but the direction is clear.