build_releases:
    GOOS=linux GOARCH=amd64 go build -o ch-linux-amd64
    GOOS=darwin GOARCH=arm64 go build -o ch-macos-arm64

test:
    go build . && echo -e "ERROR: Connection failed at 10:23:45\nWARNING: Retry attempt 1 of 3\nINFO: Connecting to database server\nSUCCESS: Connection established\nDEBUG: Query executed in 45ms\nNOTICE: Cache hit ratio: 95%\nERROR: Timeout after 30s\nSUCCESS: Backup completed" | ./ch -w -b -a error::red warning::orange info success::green ms %
