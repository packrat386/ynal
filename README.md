ynal
---

YNAL: You Need A License

Check it out at https://ynal.packrat386.com

## Development

To run `go build` then `./ynal`.

To test `go test`.

For adding licenses, add a file with the extension `.txt` to `licenses/`. The path it is served under will become the filename (with the extension trimmed) in all lowercase. For example `licenses/MIT.txt` becomes `/mit`.

## Deployment

Docker is recommended. Set `YNAL_ADDR` to tell it where to listen.

See: https://github.com/packrat386/ynal/pkgs/container/ynal
