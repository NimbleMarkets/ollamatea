Output cmd/ot-timechart/demo.gif

Require zstdcat

Set Shell "bash"
Set Padding 0
Set Framerate 40
Set FontSize 20
Set Width 1400
Set Height 600

Hide
Type "./bin/ot-timechart --in tests/SPY.2024.1109.cut.csv.zstd --title 'SPY from 2023-11-09 to 2024-11-09' --braille"
Enter
Show

# Adjust the window size for show
Type "tell me the high and low in the graph.  is it an increasing or decreasing trend?"
Enter

Sleep 10000ms

Ctrl+C

