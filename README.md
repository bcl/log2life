= log2life 

This project reads lighttpd server logs, converts the request data to [Life
1.05 patterns](https://conwaylife.com/wiki/Life_1.05) and then sends them to a
[sdl2-life server](https://github.com/bcl/sdl2-life) using the client IP as an
x, y coordinate in the Life world.

== Quickstart

* Download and build [sdl2-life server](https://github.com/bcl/sdl2-life)
* Build log2life by running `go build`
* Start the life server with `sdl2-life -rows 255 -columns 255 -server -empty`
* Pass a logfile to the server by running `log2life -width 255 -height 255 /path/to/logfile.log`

That will use the timestamps in the logfile to replay the requests in realtime.
You can control the playback speed by passing '-speed 10' to playback at 10x
realtime.

The width and height should match the rows and columns used in sdl2-life.

If you want to pipe live server logs you can do something like this:

    ssh foo@server tail -f /var/log/lighttpd/access.log | log2life -

Which will read from stdin and ignore the timestamps.
