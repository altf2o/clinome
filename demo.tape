# Record the README demo with https://github.com/charmbracelet/vhs:
#   vhs demo.tape
Output demo.gif
Set FontSize 16
Set Width 2090
Set Height 1360
Set Theme "Catppuccin Mocha"

Hide
Type "go build -o clinome . && clear" Enter
Show

Type "./clinome -bpm 100" Enter
Sleep 1s
Space
Sleep 3s
Type "lll"
Sleep 2s
Type "]"
Sleep 2s
Type "s"
Sleep 500ms
Down Down Down
Sleep 1s
Enter
Sleep 2s
Space
Sleep 500ms
Type "q"
