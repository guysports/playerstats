# Dream Team login

`playerstats login` opens Chrome or Chromium and automates The Sun's Dream Team
sign-in. It reads credentials from `~/.guysports/guysports.json`:

```json
{
  "dt_username": "your-email",
  "dt_password": "base64-encoded-password"
}
```

Base64 is encoding, not encryption. Keep the file private (for example,
`chmod 600 ~/.guysports/guysports.json`) and never commit it. An existing file
must contain both fields and a valid base64-encoded password; errors in the file
are reported rather than falling back to environment variables.

If the file does not exist, credentials are read from the environment:

```sh
export DT_USERNAME='your-email'
read -rs DT_PASSWORD
export DT_PASSWORD
go run . login
unset DT_PASSWORD
```

Enter the password when `read` waits for input. A graphical desktop and Chrome
or Chromium are required. Complete any device verification or other prompts
in the browser; `--timeout=5m` extends the default three-minute deadline.
The command does not bypass verification and may need updating if the site
changes its sign-in flow. Failed sign-in or verification can end in a timeout.

On success the command verifies the Gaming Hub access token with the fantasy
backend and saves it in a file with owner-only permissions. By default the file
is `playerstats/dreamteam-token` under the operating system's user config
directory (on macOS, `~/Library/Application Support`). Set `DT_TOKEN_FILE`
to choose a different location. Treat this token like a password.
Keep the token file outside the repository and never commit it.

Existing commands use `DT_TOKEN` if set; otherwise they read the saved file.
Unset an older `DT_TOKEN` to use the newly saved token. Other commands still
require `GSADMIN_PW` as before. Tokens expire; run `playerstats login` again
when the API rejects one. The command does not store a refresh token or refresh
the saved token automatically.
