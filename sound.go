package main

import (
    "gopkg.in/redis.v2"
    "github.com/ramzes642/sound/mixer"
    "syscall"
    "os"
    "log"
    "strings"
    "strconv"
    "os/exec"
    "path"
    "flag"
)

var AUDIO_PATH string;
var logfile string;
var redisaddr string;
var redisdb int64;
var redispass string;
var redischan string;
var channelcount int;
var daemonname string;
var cachedir string;
var samplerate int;
var channelstart int;

func GetMessage(onmsg chan string, pubsub *redis.PubSub) {

    for {
        msg, err := pubsub.Receive()
        if err != nil {
            log.Print("Redis message err: ", err)
            //panic(err)
        } else {
            //log.Print(msg.(redis.Message))
            switch v := msg.(type) {
                case *redis.Message:
                onmsg <- v.Payload
                default:
                log.Print("Unknown type:", msg)
            }
        }
    }

}

func CheckConvert(filename string, callback func(filename string)) {

    if _, err := os.Stat(AUDIO_PATH+filename); os.IsNotExist(err) {
        log.Printf("no such file: %s", filename)
        return
    }

    if filename[len(filename)-4:] == ".mp3" {
        if _, err := os.Stat(cachedir + filename+".wav"); os.IsNotExist(err) {
            go func(filename string) {
                os.MkdirAll(path.Dir(cachedir + filename), 0777)


                err := exec.Command("mpg123", "-w", cachedir + filename+".wav", "-m", "-r", strconv.Itoa(samplerate), AUDIO_PATH+filename).Run()
                if err != nil {
                    log.Print(err)
                }
                callback(cachedir + filename+".wav")
            }(filename)
        } else {
            callback(cachedir + filename+".wav")
        }

    } else {
        callback(AUDIO_PATH + filename)
    }
}

func main() {
    // Params
    flag.StringVar(&logfile, "log", "", "Log file (default stderr)")
    flag.StringVar(&AUDIO_PATH, "path", "", "Sounds directory")
    flag.IntVar(&channelcount, "channels", 6, "Channels count")
    flag.StringVar(&redisaddr, "redis", "127.0.0.1:6379", "Redis server")
    flag.StringVar(&redischan, "redischan", "rt", "Redis channel")
    flag.Int64Var(&redisdb, "redisdb", 1, "Redis db")
    flag.StringVar(&redispass, "redispass", "", "Redis password (default empty)")
    flag.StringVar(&daemonname, "name", "snd", "Daemon name")
    flag.StringVar(&cachedir, "cache", "/tmp", "Wav cache directory")
    flag.IntVar(&samplerate, "rate", 44100, "Sample rate")
    flag.IntVar(&channelstart, "chanstart", 0, "Channel start")

    flag.Parse()

    if AUDIO_PATH == "" {
        flag.PrintDefaults()
        log.Fatal("No sound path configured ("+AUDIO_PATH+")")
    }
    if AUDIO_PATH[len(AUDIO_PATH)-1:] != "/" {
        AUDIO_PATH += "/"
    }
    if cachedir[len(cachedir)-1:] != "/" {
        cachedir += "/"
    }
    // /Params

    if logfile != "" {

        file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            log.Fatalln("Failed to open log file", logfile, ":", err)
        }
        log.SetOutput(file);
    }



    log.Println("Audio daemon started")


    client := redis.NewClient(&redis.Options{
        Network: "tcp",
        Addr:     redisaddr,
        Password: redispass, // no password set
        DB:       redisdb, // use default DB
    })

    pubsub := client.PubSub()

    err := pubsub.Subscribe(redischan)
    if err != nil {
        log.Fatal("Redis subscribe: ", err)
    }
    defer pubsub.Close()

    msgEvent := make(chan string)

    go GetMessage(msgEvent, pubsub)

    mix := mixer.NewChannelsMixer(channelcount)

    chanData := mix.Mix()
    defer mix.Close()

    Stdout := os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")

    sampleBuffer := make([]byte, 0)
    dataPos := 0
    for {
        select {
        case data := <-chanData:
            sampleBuffer = append(sampleBuffer, data...)
            dataPos++
            if dataPos == 100 {
                Stdout.Write(sampleBuffer)
                dataPos = 0
                sampleBuffer = make([]byte, 0)
            }

        case msg := <-msgEvent:
            cmd := strings.SplitN(msg, ":", 2)

            if cmd[0] == "u-snd" { // channel u-snd
                log.Printf("Command: %s\n", msg)

                cmd = strings.Split(cmd[1], ":")
                switch cmd[0] {
                    case "sfx":
                    // (location,volume,filename) = data.split(':',2)
                    channels := strings.Split(cmd[1], ",")
                    for _, c := range channels {
                        vol, _ := strconv.ParseFloat(cmd[2], 32)
                        cid, _ := strconv.Atoi(c)
                        cid -= channelstart
                        if cid > 0 && cid < channelcount {
                            log.Printf("Adding sfx to %d\n", cid)
                            CheckConvert(cmd[3], func(filename string) {
                                mix.GetChannel(cid-1).AddSound(filename, mixer.WT_SFX, vol)
                            })
                        }

                    }

                    case "loop":
                    channels := strings.Split(cmd[1], ",")
                    for _, c := range channels {
                        vol, _ := strconv.ParseFloat(cmd[2], 32)
                        cid, _ := strconv.Atoi(c)
                        cid -= channelstart
                        if cid > 0 && cid < channelcount {
                            log.Printf("Adding loop to %d\n", cid)
                            CheckConvert(cmd[3], func(filename string) {
                                mix.GetChannel(cid-1).StopSound(mixer.WT_LOOP)
                                mix.GetChannel(cid-1).AddSound(filename, mixer.WT_LOOP, vol)
                            })
                        }
                    }

                    case "stop-sfx":
                    channels := strings.Split(cmd[1], ",")
                    for _, c := range channels {
                        cid, _ := strconv.Atoi(c)
                        cid -= channelstart
                        if cid > 0 && cid < channelcount {
                            mix.GetChannel(cid-1).StopSound(mixer.WT_SFX)
                        }
                    }

                    case "stop-all":
                    channels := strings.Split(cmd[1], ",")
                    for _, c := range channels {
                        cid, _ := strconv.Atoi(c)
                        cid -= channelstart
                        if cid > 0 && cid < channelcount {
                            mix.GetChannel(cid-1).StopSound(mixer.WT_ALL)
                        }
                    }
                    case "fade-all":
                    channels := strings.Split(cmd[1], ",")
                    fadeSpeed, _ := strconv.ParseFloat(cmd[2], 32)
                    for _, c := range channels {
                        cid, _ := strconv.Atoi(c)
                        cid -= channelstart
                        if cid > 0 && cid < channelcount {
                            mix.GetChannel(cid-1).FadeSound(mixer.WT_ALL, fadeSpeed)
                        }
                    }
                }
            }
            if cmd[0] == "u-ping" { // channel u-snd
                client.Publish("rt", "u-pong:"+daemonname+":ok")
            }
        }
    }
}

