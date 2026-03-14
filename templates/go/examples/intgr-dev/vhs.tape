Require dtkt

Set Shell "bash"
Set FontFamily "IBM Plex Mono"
Set FontSize 16
Set Width 1000
Set Height 600

Hide
Type `sh -c 'sleep 5 && touch example-file.txt && sleep 3 && rm example-file.txt' &` Enter
Type `clear` Enter
Show

Type `dtkt intgr dev` Enter@500ms

Sleep 5s

# File created

Sleep 3s

# File deleted

Sleep 3s

Ctrl+C

Sleep 2s
