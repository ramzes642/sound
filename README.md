# soundmixer
This is a network audio player for smart home.

## Quick start

Install:
    
    $ go get github.com/ramzes642/sound
    $ go build github.com/ramzes642/sound

Run server:
	
	$ $GOPATH/bin/sound -path /Users/ramzes/tmp -redis 127.0.0.1:6379 | play -c 6 -b 16 -e signed -t raw -r 44.1k -q -

Send sound to play:
	
	$ redis-cli publish rt u-snd:sfx:1:100:./audio/hints/danger/no.wav

## Play commands

	u-snd:<type>:<channel>:<volume>:<filename>

types:
* sfx - play once
* loop - repeat
	
channel: 
* Desired channel number 1-6, or comma list (ex. 1,2,3)
	 
volume:
* 0 - 100 - in percent
* -1 - -∞  - in Dbm

Filename:
* file to play wav or mp3 (uses mpg123 to reconvert)
	
## Other commands

Stop:

	u-snd:stop-sfx:<channel>
	u-snd:stop-all:<channel>
	
Fade:
	
	u-snd:fade-all:<channel>


## Command-line arguments
```sh
  -cache string
    	Wav cache directory (default "/tmp")
  -channels int
    	Channels count (default 6)
  -chanstart int
    	Channel start
  -log string
    	Log file (default stderr)
  -name string
    	Daemon name (default "snd")
  -path string
    	Sounds directory
  -rate int
    	Sample rate (default 44100)
  -redis string
    	Redis server (default "127.0.0.1:6379")
  -redischan string
    	Redis channel (default "rt")
  -redisdb int
    	Redis db (default 1)
  -redispass string
    	Redis password (default empty)
```
