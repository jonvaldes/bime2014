package main

import (
	"code.google.com/p/portaudio-go/portaudio"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

var stream *portaudio.Stream

type Handler struct{}

var index = `
<html>
<body>

</body>
<script>
    var userID = Date.now();
    var beta = 0.0;
    var gamma = 0.0;

	window.addEventListener('deviceorientation', function (event) {
		gamma = event.gamma;
		beta = event.beta;

		if(navigator.isCocoonJS){
			gamma = event.gamma;
			beta = event.beta;

			beta *= 10;
		}
	});

/*
 	window.addEventListener('devicemotion', function (event) {
		beta = event.accelerationIncludingGravity.y;
	});*/

	var send = function(){
		var oReq = new XMLHttpRequest();
		oReq.open("get", "/move?userID="+userID +"&gamma="+gamma+"&beta="+beta, true);
		oReq.send();
		setTimeout(send, 100);
	};

	send();
</script>
</html>`

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/favicon.ico" {
		return
	}

	if r.URL.Path == "/" {
		fmt.Fprint(w, index)
		return
	}
	fmt.Println(r.URL.RawQuery)
	query := r.URL.Query()

	sgamma := query.Get("gamma")
	sbeta := query.Get("beta")

	gamma, _ := strconv.ParseFloat(sgamma, 64)
	if gamma > 90 {
		gamma -= 360
	}
	beta, _ := strconv.ParseFloat(sbeta, 64)

	userID, _ := strconv.Atoi(query.Get("userID"))

	found := false
	for i := 0; i < len(playingSamples); i++ {
		if playingSamples[i].userID == userID {
			playingSamples[i].rate = 1.0 + float32(gamma/180.0)
			playingSamples[i].volume = 0.5 + float32(beta*0.02)
			if playingSamples[i].volume < 0.0 {
				playingSamples[i].volume = 0.0
			}
			if playingSamples[i].volume > 2.0 {
				playingSamples[i].volume = 2.0
			}
			fmt.Println("rate:", playingSamples[i].rate, "volume:", playingSamples[i].volume)
			found = true
			break
		}
	}

	if !found {
		sample := getSampleForUser(userID)
		playingSamples = append(playingSamples, sample)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

type SampleData struct {
	userID  int
	data    []float32
	playPos float32
	volume  float32
	rate    float32
}

var filesData = map[string][]float32{}

var files = []string{"test1.raw", "test2.raw", "test3.raw", "test4.raw"}

var nextUser = rand.Intn(len(files))

func getSampleForUser(userID int) SampleData {

	filename := files[nextUser%len(files)]
	nextUser++

	// Open the file.
	var data []float32
	data, ok := filesData[filename]
	if !ok {
		var err error
		f, err := os.Open(filename)
		stat, _ := f.Stat()
		length := stat.Size()
		data = make([]float32, length/4)
		chk(err)
		chk(binary.Read(f, binary.LittleEndian, &data))

		filesData[filename] = data
	}

	return SampleData{userID, data, 0, 1, 1.0}
}

func main() {
	runtime.GOMAXPROCS(4)
	portaudio.Initialize()
	defer portaudio.Terminate()

	var err error
	stream, err = portaudio.OpenDefaultStream(0, 2, 44100, 0, processAudio)
	if err != nil {
		panic(err)
	}
	defer stream.Close()
	chk(stream.Start())

	//playingSamples = append(playingSamples, getSampleForUser(0))
	/*playingSamples = append(playingSamples, getSampleForUser(0))
	playingSamples[1].playPos = 44100
	playingSamples[1].rate = 1.3*/

	fmt.Println("Play!")

	log.Fatal(http.ListenAndServe(":8080", Handler{}))
}

var playingSamples = []SampleData{}
var samplesMutex sync.Mutex

func processAudio(out [][]float32) {

	for i := range out[0] {

		var value float32

		for j := range playingSamples {
			value += playingSamples[j].volume * playingSamples[j].data[int(playingSamples[j].playPos)%len(playingSamples[j].data)]
			playingSamples[j].playPos += playingSamples[j].rate
		}
		out[0][i] = value
		out[1][i] = value
	}

	for i := range playingSamples {
		playingSamples[i].volume *= 0.99
	}

	//samplesMutex.Unlock()
}
