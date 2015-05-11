
# go-sparkline

![](demo.gif)

## Install
``` shell
# install via go tools
go get github.com/muhqu/go-sparkline
go install github.com/muhqu/go-sparkline

# verify
go-sparkline --help
```

## Help

```
$ go-sparkline --help
usage: go-sparkline [<flags>] [<values>]

Flags:
  --help             Show help.
  -s, --stream       stream
  --animate          start animation
  --lazy             ignore parse errors
  --char-size=7:17   Pixel size of a single character. Can also be set via env
                     ITERM_CHARACTER_SIZE. The default 7:17 corresponds to 12p Monaco.
  --rows=3           height in number of rows
  --renderer=sparks  available renderers: line, sparks, vlines

Args:
  [<values>]  Numeric values to render. Can also be read from stdin.
```

## Examples

### simple numbers
```
go-sparkline 16 19 18 12 7 4 7 15 25 33 35 32 26 21
# or
echo 16 19 18 12 7 4 7 15 25 33 35 32 26 21 | go-sparkline
```

### simple json array
```
echo '[16,19,18,12,7,4,7,15,25,33,35,32,26,21]' | go-sparkline
```

###  streamed json arrays
```
(
echo '[16,19,18]'; sleep 1;
echo '[12,7,4]'; sleep 1;
echo '[7,15,25]'; sleep 1;
echo '[33,35,32]'; sleep 1;
echo '[26,21]';
) | go-sparkline --stream
```

### streamed ping graph
```
ping -n -i 0.3 localhost \
 | awk 'BEGIN{FS="time=|ms"}/time=/{printf "%d\n",$2*1000;fflush()}' \
 | go-sparkline --stream
```

### aws cloudwatch metric data
```
$ aws cloudwatch get-metric-statistics \
    --namespace AWS/ELB \
    --metric-name RequestCount \
    --end-time "2015-04-15T21:15:00Z" \
    --start-time "2015-04-15T17:45:00Z" \
    --period 900 \
    --statistics Sum \
    | tee cloudwatch.json \
    | go-sparkline
$ head cloudwatch.json; echo '...'; tail cloudwatch.json;
{
    "Datapoints": [
        {
            "Timestamp": "2015-04-15T17:45:00Z",
            "Sum": 37823.0,
            "Unit": "Count"
        },
        {
            "Timestamp": "2015-04-15T11:30:00Z",
            "Sum": 12413.0,
...
            "Unit": "Count"
        },
        {
            "Timestamp": "2015-04-15T21:15:00Z",
            "Sum": 29428.0,
            "Unit": "Count"
        }
    ],
    "Label": "RequestCount"
}
```


# FAQ

## Q: What?! How does it display inline images directly in the terminal?!

**A:** [iTerm2][] supports a bunch of [propritary ESC seq][iTerm2seq]. The [nightly build][iTerm2nightly] even includes one to [render images][iTerm2images] directly into the terminal. 
```
ESC ] 1337 ; File = [optional arguments] : base-64 encoded file contents ^G
```


[iTerm2]: http://iterm2.com/
[iTerm2seq]: http://iterm2.com/documentation-escape-codes.html
[iTerm2images]: http://iterm2.com/images.html
[iTerm2nightly]: http://iterm2.com/downloads/nightly/

# Autor


|   |   |
|---|---|
| ![](http://gravatar.com/avatar/0ad964bc2b83e0977d8f70816eda1c70) | © 2015 by Mathias Leppich <br>  [github.com/muhqu](https://github.com/muhqu), [@muhqu](http://twitter.com/muhqu) |
|   |   |

