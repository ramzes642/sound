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

    return mixer
}

func (this ChannelsMixer) Close() chan bool {
    log.Fatal("Mixer closing")
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
                    }

                }
            dataChan <- data
            }
        }
    }()

    return dataChan
}