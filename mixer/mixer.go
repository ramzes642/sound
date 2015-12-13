package mixer
import (
    "log"
    "time"
)


type ChannelsMixer struct {
    channelsCount int
    channels []*Channel
    closing chan chan bool
}


func NewChannelsMixer (count int) ChannelsMixer {

    mixer := ChannelsMixer{count, make([]*Channel, count), make(chan chan bool) }


    for i:=0; i<count; i++ {
        mixer.channels[i] = &Channel{
            volume: 1,
            dataChan: make(chan [2]byte, buf_size),
            waves: make(chan []*Wave, 1),
            closing: make(chan chan string,1),
        }
        mixer.channels[i].waves <- []*Wave{} // init empty array in channel buffer
    }
    log.Printf("channels: %d\n",len(mixer.channels))

//    mixer.channels[0].AddSound("test1.wav",WT_SFX,60)
//    mixer.channels[0].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[0].AddSound("test3.wav",WT_LOOP,0.5)
//    mixer.channels[1].AddSound("test1.wav",WT_LOOP,0.5)
//    mixer.channels[1].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[1].AddSound("test3.wav",WT_LOOP,0.5)
//    mixer.channels[2].AddSound("test1.wav",WT_LOOP,0.5)
//    mixer.channels[2].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[2].AddSound("test3.wav",WT_LOOP,0.5)
//    mixer.channels[3].AddSound("test1.wav",WT_LOOP,0.5)
//    mixer.channels[3].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[3].AddSound("test3.wav",WT_LOOP,0.5)
//    mixer.channels[4].AddSound("test1.wav",WT_LOOP,0.5)
//    mixer.channels[4].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[4].AddSound("test3.wav",WT_LOOP,0.5)
//    mixer.channels[5].AddSound("test1.wav",WT_LOOP,0.5)
//    mixer.channels[5].AddSound("test2.wav",WT_LOOP,0.5)
//    mixer.channels[5].AddSound("test3.wav",WT_LOOP,0.5)

    return mixer
}

func (this ChannelsMixer) Close() chan bool {
    log.Printf("Mixer closing\n")
    panic("closing")
    closed := make(chan bool)
    this.closing <- closed
    return closed
}

func (this ChannelsMixer) GetChannel(id int) *Channel {
    if this.channels[id] != nil {
        return this.channels[id]
    }
    return nil
}

func (this ChannelsMixer) Mix() chan []byte {

    dataChan := make(chan []byte)

    chanCount := len(this.channels)
    log.Printf("Channels count %d",chanCount)

    // start channels mixing
    for i:=0; i < chanCount; i++ {
        go this.channels[i].render()
    }


    // start composition
    go func() {
        var closer chan string
        var pcs [2]byte

        for {
            select {
            case closed := <-this.closing:
                log.Printf("Channels closing")
                closer = make(chan string,chanCount)
                for i := 0; i < chanCount; i++ {
                    log.Printf("Channel %d closing",i)
                    this.channels[i].closing <- closer
                    loop:
                    for {
                        select {
                        case msg := <-closer:
                            log.Printf("Channel %d closed: %s", i, msg)
                            break loop
                        case <-this.channels[i].dataChan:
                        case <-time.After(time.Second):
                            log.Printf("Channel %d timeout", i)
                        }
                    }
                }
                closed <- true
                return

            default:
                data := make([]byte,chanCount*2)

                for i := 0; i < chanCount; i++ {

                    select {
                    case pcs = <-this.channels[i].dataChan:
                        data[i*2] = pcs[0]
                        data[i*2 + 1] = pcs[1]
                    //binary.LittleEndian.PutUint16(data)
                    //binary.Write(data, binary.LittleEndian, pcs)
                    //                        case <-time.After(time.Second):
                    //                            log.Printf("Composer timeout")
                    }


                    //binary.BigEndian.PutUint16(data, )
                }
            dataChan <- data
//                    select {
//                    case dataChan <- data:
//                    case <-time.After(time.Second):
//                        log.Printf("Write timeout")
//                        panic("Write timeout (race condition)")
//                    }
            }
        }
    }()

    return dataChan
}