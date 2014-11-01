package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

type Handler struct{}

var index = `
<html>
<body>

</body>
<script>
    var userID = Date.now();
    var beta = 0.0;
    var gamma = 0.0;

	var smooth = 0.01;
	window.addEventListener('deviceorientation', function (event) {
		
	 	gamma = event.gamma;
        beta = event.beta;
	});

	var send = function(){
		var oReq = new XMLHttpRequest();
		oReq.open("get", "/move?userID="+userID +"&gamma="+gamma+"&beta="+beta, true);
		oReq.send();
		//setTimeout(send, 200);
	};

	//send();

 	window.addEventListener('touchstart', function (event) {
 		send();
 	});

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
	beta, _ := strconv.ParseFloat(sbeta, 64)

	userID, _ := strconv.Atoi(query.Get("userID"))

	if _, ok := users[userID]; !ok {
		users[userID] = initUser(userID)
	}

	note := int(10 + float32((1.0+gamma/180.0)*64))
	volume := int(180.0 * float32(beta*0.01))
	if volume > 127 {
		volume = 127
	}
	fmt.Fprintf(users[userID].outFile, "%d %d %d\n", userID, note, volume)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	runtime.GOMAXPROCS(4)

	log.Fatal(http.ListenAndServe(":8080", Handler{}))
}

var users = map[int]UserData{}

type UserData struct {
	outFile *os.File
	userID  int
	command *exec.Cmd
}

func initUser(userID int) UserData {
	filename := fmt.Sprintf("%d.txt", userID)

	os.Remove(filename)
	dataFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	chk(err)

	exec.Command("python", "createInstrument.py", strconv.Itoa(userID)).Run()
	command := exec.Command("python", "generateSound.py", filename)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	go func() {
		for {
			command.Start()
			command.Wait()
		}
	}()

	return UserData{dataFile, userID, command}
}
