Due to issues with supporting CBC, this project chooses to currently use the forks at
https://github.com/ScriptRock/crypto and https://github.com/ScriptRock/sftp. See
https://groups.google.com/forum/#!topic/golang-nuts/J2XCsTsNQ9o.

This would be on the normal GOPATH but they broke the Windows build in upstream SFTP, see:
https://github.com/pkg/sftp/issues/47. Since https://github.com/ScriptRock/sftp hasn't merged that yet, we just did it
for him in our own fork until he gets around to it. Instead of hosting the fork on public internet, I decided
temporarily to just bury it in here using the Golang vendor experiment since it only affects Windows builds.