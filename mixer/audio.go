package mixer
import (
    "log"
    "os"
    "io"
    "math"
    "encoding/binary"
    "time"
)

type WaveType byte

const (
    WT_SFX WaveType = iota
    WT_LOOP
    WT_ALL
)

const buf_size = 100
const read_buf_size = 65536

type Wave struct {
    file *os.File
    wtype WaveType
    volume, initVolume float64
    buffer []byte
    pos, end int
    read, maxpos uint32
    stereo bool
}

type Channel struct {
    volume float64
    dataChan chan [2]byte
    waves chan []*Wave
    closing chan chan string
}


func (this *Channel) AddSound(filename string, wtype WaveType, volume float64) {
    var file *os.File;
    file, err := os.Open(filename) // For read access.
    if err != nil || file == nil {
        log.Printf("\nAddsfx error %s\n",err.Error())
        return
    }
    if volume <= 0 {
        volume = math.Pow(10, volume / 10) // dbm mode
    } else {
        volume = volume / 100 // percent mode
    }

    waves := <- this.waves // get array
    w := &Wave{
        file: file,
        wtype: wtype,
        volume: volume,
        initVolume: volume,
        buffer: make([]byte, read_buf_size),
        pos: 0,
        read: 0,
    }
    waves = append(waves, w)
    w.open()
    this.waves <- waves // return array

}

func (this *Channel) StopSound(wtype WaveType) {
    waves := <- this.waves // get array
    if len(waves) > 0 {
        if wtype == WT_ALL {
            waves = []*Wave{}
        } else {
            for i := 0; i<len(waves); i++ {
                if waves[i].wtype == wtype {
                    waves = append(waves[:i], waves[i+1:]...)
                    i--
                }
            }
        }
    }
    this.waves <- waves // return array
}

func (this *Channel) FadeSound(wtype WaveType, fadeSpeed float64) {
    go func(speed float64) {
        var i float64 = 1.0
        for  ; i>0; i -= speed {
            waves := <-this.waves // get array{
            for wid, w := range waves {
                if wtype == WT_ALL || w.wtype == wtype {
                    waves[wid].volume = w.initVolume * i
                }
            }

            this.waves <- waves // return array
            time.Sleep(time.Millisecond * 10)
        }
        log.Printf("Faded")
        this.StopSound(wtype)

    }(1/fadeSpeed/100)
}


func (this *Wave) open() error {

    this.file.Seek(0,0)
    this.buffer = make([]byte, read_buf_size)
    c, err := this.file.Read(this.buffer)
    if c > 0 || err != nil {
        this.end = c
    }
    // TODO: Wave channels detection
    this.stereo = false
    this.pos = 48
    this.read = 48
    subchunkSizeOffset := binary.LittleEndian.Uint32(this.buffer[16:20]) + 24
    this.maxpos = binary.LittleEndian.Uint32(this.buffer[subchunkSizeOffset:subchunkSizeOffset+4])

    return err
}

func (this *Wave) ReadInt16() (int16, error) {
    var result int16 = 0

    result = int16(this.buffer[this.pos]) | int16(this.buffer[this.pos+1])<<8

    if this.end == 0 {
        return 0, io.EOF
    }


    if this.stereo {
        this.pos+=4
        this.read+=4
    } else {
        this.pos+=2
        this.read+=2
    }


    if this.read >= this.maxpos { // got wave end
        return 0, io.EOF
    }

    if this.pos >= this.end {
        if this.end == 0 {
            log.Printf("Read 0: pos: %d\n",this.pos)
        }
        this.pos = this.pos % this.end // rewind buffer pos

        this.buffer = make([]byte, read_buf_size)
        c, err := this.file.Read(this.buffer)
        if c > 0 || err != nil {
            this.end = c
           // log.Printf("Read: %d\n",c)
        }
        if this.pos >= this.end {
            return 0, io.EOF
        }

    }

    return result, nil
}

func (this *Channel) render() {
    //var chunk int16;

    for {
        select {
        case result:= <-this.closing:
            waves := <-this.waves
            for _, w := range waves {
                w.file.Close()
            }
            this.waves <- []*Wave{} // return empty array
            result <- "Channel closed"
            return

        default:
            waves := <-this.waves // get array
            if len(waves) == 0 {
                //log.Fatal(this.waves)
                this.waves <- waves // return array
                this.dataChan <- [2]byte{0,0}
            } else {
                var waveChanByte int16 = 0


                for i, w := range waves {
                    chunk, err := w.ReadInt16()
                    if err == io.EOF {
                        if w.wtype == WT_SFX {
                            w.file.Seek(0, 0)
                            waves = append(waves[:i], waves[i+1:]...)

                        } else if w.wtype == WT_LOOP {
                            w.open()
                        }
                        //fmt.Println(err)
                    } else {
                        //binary.Read(waveData, binary.LittleEndian, &chunk)
                        waveChanByte += int16(float64(chunk) * w.volume)

                        //w.file.Seek(2, 1)
                    }
                }
                this.waves <- waves // return array


                //log.Printf("%d",waveChanByte)
                select {
                case this.dataChan <- [2]byte{ byte(waveChanByte), byte(waveChanByte  >> 8) }:
                case result:= <-this.closing:
                    waves := <-this.waves // get array
                    for _, w := range waves {
                        w.file.Close()
                    }
                    this.waves <- []*Wave{} // return empty array

                    result <- "Channel closed"
                    return
                }
            }
        }
    }
}
