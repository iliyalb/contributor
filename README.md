# Contributor

A rewrite of [github-activity-generator](https://github.com/Shpota/github-activity-generator) in go

why?
Go is faster than Python, there is no need for Python interpreter, it uses lower memory, has error checking, and the compile-time type checking prevents runtime errors

## Usage

Build the program
```sh
go build -o github-activity-generator
```

Basic usage with remote repository
```sh
./github-activity-generator -r https://github.com/user/repo.git -un "John Doe" -ue john@example.com
```

Skip weekends, 60% frequency, 180 days back, max 5 commits per day
```sh
./github-activity-generator -nw -fr 60 -db 180 -mc 5
```

Use long form flags
```sh
./github-activity-generator --repository https://github.com/user/repo.git --no_weekends --frequency 50
```

## Command-line Options

- `-r, --repository`: Remote git repository URL
- `-un, --user_name`: Git user name
- `-ue, --user_email`: Git user email
- `-nw, --no_weekends`: Skip weekends
- `-fr, --frequency`: Percentage of days to commit (0-100)
- `-db, --days_before`: Days before current date to start
- `-da, --days_after`: Days after current date to end
- `-mc, --max_commits`: Maximum commits per day (1-20)

## LICENSE

This is free and unencumbered software released into the public domain under The Unlicense.

Anyone is free to copy, modify, publish, use, compile, sell, or distribute this software, either in source code form or as a compiled binary, for any purpose, commercial or non-commercial, and by any means.

See [UNLICENSE](LICENSE) for full details.
