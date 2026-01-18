package main

import (
	"fmt"
	"os"
	"log"
	"bufio"
	"io"
	"time"
	"net"
	"net/http"
	"bytes"
)
type dattim struct {	//тип данных макс./мин. за всё время мониторинга
		D [3][5][]float64 //данные IUP ([3][5][] ток, напряжение, мощность)
		T [3][5][]string //моменты времени их регистрации
	}
	var Max = dattim {[3][5][]float64 {{{0,0,0}, {0,0,0}, {0,0,0}, {0}, {0}},
			  {{0,0,0}, {0,0,0}, {0}, {0}, {0}},
			  {{0,0,0}, {0}, {0}, {0}, {0,0,0}}},
			[3][5][]string{{{"0","0","0"}, {"0","0","0"}, {"0","0","0"}, {"0"}, {"0"}},
			 {{"0","0","0"}, {"0","0","0"}, {"0"}, {"0"}, {"0"}},
			 {{"0","0","0"}, {"0"}, {"0"}, {"0"}, {"0","0","0"}}}}

	var Min = dattim {[3][5][]float64 {{{65000,65000,65000}, {65000,65000,65000}, {65000,65000,65000}, {65000}, {65000}},
			  {{65000,65000,65000}, {65000,65000,65000}, {65000}, {65000}, {65000}},
			  {{65000,65000,65000}, {65000}, {65000}, {65000}, {65000,65000,65000}}},
		[3][5][]string{{{"0","0","0"}, {"0","0","0"}, {"0","0","0"}, {"0"}, {"0"}},
			 {{"0","0","0"}, {"0","0","0"}, {"0"}, {"0"}, {"0"}},
			 {{"0","0","0"}, {"0"}, {"0"}, {"0"}, {"0","0","0"}}}}
type point struct {
	t string
	d [3][5]string
	}
	//токи:текущ., средн., макс. (по фазам), нул. и обр. последовательности
	//напряжения: фазное,линейное (по фазам), прямая, обратная, нулевая последовательности
	//мощности  : активная (по фазам), активная полная, реактивная, суммарная полная, косинус фи (по фазам)
	var td = point {"",
			[3][5]string {{"0 0 0", "0 0 0", "0 0 0", "0", "0"}, //токи
			  		 {"0 0 0", "0 0 0", "0", "0", "0"},	//напряжения
			  		 {"0 0 0", "0", "0", "0", "0 0 0"}}}	//мощности
			
	var std []point
	var st0 string	//время начала мониторинга
	var namr = [][]string {{"Токи фаз", "Cредние значения за минуту", "Пиковые значения ",
			"Нулевая последовательность", "Обратная последовательность"}, 
			{"Напряжения фаз", "Линейные напряжения", "Прямая последовательность", 
			"Обратная последовательность", "Нулевая последовательность" },
			{"Активная по фазам", "Суммарная активная", "Реактивная",
			"Полная", "Косинус фи по фазам"}}
func main() {
	var fd *os.File
	var err error
	var st, sd string
	var s []string
	begt, endt, ipadr, err := parFromCommand()
	if err != nil {
	fmt.Println (err)
	//return //
	}
	hipport := ipadr + ":8183" 
	if fd, err = os.Open("om310dat.txt"); err != nil {
	log.Fatal(err)
	}
	defer fd.Close()
	r := bufio.NewReader(fd)
	st, err = r.ReadString('\n')	//первая строка - время начала мониторинга (18:21:10)
	st0 = st[:(len(st)-1)] //убираем '\n'  
	//читаем весь файл и формируем срез std
	for err != io.EOF {
	st, err = r.ReadString('\n')
	if len (st) == 0 { break }
	if st[0] != '['  {break} //last string test
	s = spl(st, '[',']')
	xts := tnw(s[0])	// time in sec
		if xts < begt  { 
		_, err = r.ReadString('\n')
		_, err = r.ReadString('\n')
		_, err = r.ReadString('\n')
		_, err = r.ReadString('\n')
		continue
		}
	if xts > endt { break }
	td.t = s[0] //момент считывания данных из ОМ310
		for i :=0; i<3 ; i++ {
		sd, err = r.ReadString('\n') //читаем 3 строки (токи, напряжения, мощности)
		s = spl(sd, '[',']')   //разбивка на 5 подстрок
			for j, x := range(s) {
			td.d[i][j] = x
			drmaxmin(i, j, x)
			}
		}
	_, err = r.ReadString('\n')	//считываем концевик "]\n"
	std = append (std, td)
	}
	_, err = r.ReadString('\n') //for EOF test
	for err != io.EOF {
	st, err = r.ReadString('\n')
	_, err = r.ReadString('\n') //for EOF detect
	}
	//------  макс и мин  по срезу std
	for i :=0; i <3; i++ {
			for j :=0; j <5; j++ {
fmt.Printf("%s max %.1f:%s min %.1f:%s\n", namr[i][j], Max.D[i][j], Max.T[i][j], Min.D[i][j], Min.T[i][j])
			}
			}
	fmt.Printf("%s", st)
	//------- 
	http.HandleFunc("/", httphandler)
	http.HandleFunc("/pf3", plotp3)
	http.HandleFunc("/uf3", plotuf3)
	http.HandleFunc("/if3", ploti3a)
	http.HandleFunc("/cosfi", plotcosfi)
//	tend, ipadr, _ := parFromCommand()
	//hipport := ipadr + ":8183" 
     	hlistener, err := net.Listen("tcp", hipport)
		if err != nil {
        	log.Println("error starting net.Listen: ", err)			
        	return
    		} 
	defer hlistener.Close()
	go http.Serve(hlistener, nil)
	fmt.Println ("http сервер ждёт обращения на ", hipport, "(/, /pf3, /uf3, /if3, /cosfi)")	
	time.Sleep(time.Second * 30000)  //ждём обращения клиента
}
//разбиение строки по разделителям с1 и с2
func spl(s string, c1, c2 byte) ([]string) {
	var i0, i1, i int
	var xs []string
	var b byte
	ls := len(s)
	if s[(ls-1)] == '\n' {s = s[:(ls-1)]}	// отсекаем конец строки ('\n')
	//fmt.Println (len(s),s)
	bs := []byte(s)
	lbs := lencat(bs)
	bs = bs[:lbs]
	for i, b = range bs {
		if b == c1 {
			i0 = i+1
			continue
		}
		if b == c2 {
			i1 = i
			xs = append(xs, string(bs[i0:i1]))
			i0 = i1+1
			i1 = 0
		}
	if c1 == ' ' {c1 = 0}	//выключаем первый разделитель
	}
	if i1 == 0 {
	if i0 < lbs { xs = append(xs, string(bs[i0:]))} 
	}
	return xs
}
//отсечение хвостовых пробелов
func lencat(b []byte) (int) {
	n0 := len(b)
	n := n0
	for i:=0; i<n0; i++ {
		if b[n-1] == ' ' {
	 	n = n-1
		continue
		}
	break
	}
	return n
}
// получение срезов iup из их накопленных строк
func omdat(xs string) []uint16 {
	var s []string
	var rx []uint16
	var x uint16
	s = spl(xs, ' ', ' ')
	for _, sx := range (s) {
	fmt.Sscanf(sx, "%d", &x)
	rx = append (rx, x)
	}
	return rx
}
func tnw(s string) int {
	//fmt.Printf("%s\n", s)
	t, _:= time.ParseDuration(s)
	//ss := fmt.Sprintf("%.0f", t.Seconds()) //интервал округляем до целых секунд!
	//is, _ := strconv.Atoi(ss)
	it := int(t.Seconds())
	return it
}
func drmaxmin(i int, j int, sx string) {
	r := omdat(sx)
	for k, _ := range r {
	x := valf(i, j, k, r)
	if x > Max.D[i][j][k] {
		Max.D[i][j][k] = x
		Max.T[i][j][k] = td.t
		}
	if x < Min.D[i][j][k] {
		Min.D[i][j][k] = x
		Min.T[i][j][k] = td.t
		}
	}
}
// получение физической величины тока (i=0), напряжения (i=1), мощности (i=2) из среза Dfom ([]uint16)
func valf(i int, j int, k int, x []uint16) float64 {
	fk := 1. 
	if i == 0 { fk = 0.1}
	if i == 2 {
		 fk = 0.01 
			if j == 4 { fk = 0.001}
		}
	fx := float64(x[k]) * fk
	return fx
}

func httphandler(w http.ResponseWriter, r *http.Request) {
	var bufs bytes.Buffer //буфер выходной строки для передачи ответа клиенту
//	var tnow string // время съёма данных из ОМ310
//-- Измеряемые величины ОМ310
//	var iF, iS, iN, iF0, ioP om.Dfom  // фазные токи, средние, максимальные (по фазам), нулевой и обратной последовательности 
//	var UF, UL, UpP, UoP, UnP om.Dfom // фазные и линейные напряжения, прямая, обратная и нулевая поледоватеоьность
//	var Pot, PoA, PoJ, PA, PC om.Dfom // мощности полная, активная, реактивная, активная по фазам, косинус фи по фазам
//	var frq om.Dfom //частота тока
//	var twrk, pwh om.Dfom //время работы и активная электроэнергия по фазам
//--  
	//--- конец чтения и перевода в нормальные единицы
	// HTML разметка
	bufs.WriteString (`<table border="1">
	<tr><th colspan="1"> Токи (амперы)</th><th colspan="1"> Максимумы</th><th colspan="1"> Время максимумов</th>
			<th colspan="1"> Минимумы</th><th colspan="1"> Время минимумов</th></tr>`)
	for j :=0; j <5; j++ {
	bufs.WriteString (hformat(namr[0][j], Max.D[0][j], Max.T[0][j], Min.D[0][j], Min.T[0][j]))
	}
	bufs.WriteString (`</table>`)
	//-
	bufs.WriteString (`<table border="1">
	<tr><th colspan="1"> Напряжения (вольты)</th><th colspan="1"> Максимумы</th><th colspan="1"> Время максимумов</th>
			<th colspan="1"> Минимумы</th><th colspan="1"> Время минимумов</th></tr>`)
	for j :=0; j <5; j++ {
	bufs.WriteString (hformat(namr[1][j], Max.D[1][j], Max.T[1][j], Min.D[1][j], Min.T[1][j]))
	}
	bufs.WriteString (`</table>`)
	//-
	bufs.WriteString (`<table border="1">
	<tr><th colspan="1"> Мощности (квт/ква)</th><th colspan="1"> Максимумы</th><th colspan="1"> Время максимумов</th>
			<th colspan="1"> Минимумы</th><th colspan="1"> Время минимумов</th></tr>`)
	for j :=0; j <5; j++ {
	bufs.WriteString (hformat(namr[2][j], Max.D[2][j], Max.T[2][j], Min.D[2][j], Min.T[2][j]))
	}
	bufs.WriteString (`</table>`)
	//-
		//bufs.WriteString (hformat(" Потребление (квт/час по фазам): ", rpwh))
	//bufs.WriteString (hformat(" Частота тока (гц):", rfrq))
	//bufs.WriteString(hformat("Время считывания из OM310:",tnow))
	//bufs.WriteString(hformat("Время работы  OM310 (часы):",twrk))
	bufs.WriteString (`</table>`)
	// конец разметки
	bufs.WriteString(fmt.Sprintf("Старт мониторинга [%s], до последней точки [%s], снято точек [%d]", st0, td.t, len(std))) 
	fmt.Fprint (w, bufs.String()) // в ответ клиенту
}
func hformat(s string, mad []float64, mat []string, mid []float64, mit []string ) string {
	str := "<tr><td>%s</td>"
		str += "<td>%.1f</<td>"
		str += "<td>%s</td>"
		str += "<td>%.1f</<td>"
		str += "<td>%s</td>"
		str += "</tr>"
	return fmt.Sprintf(str, s, mad, mat, mid, mit)
}
func parFromCommand() (tb int, te int, ipadr string, err error) {
	ipadr = "127.0.0.1"
	tb = 0 // begin t
	te = 3600 // end t 3600sec
	err = nil
	if len(os.Args) > 1 {
		tb = tnw(os.Args[1])
		if len(os.Args) > 2 { te = tnw(os.Args[2])}
		if len(os.Args) > 3 { ipadr = os.Args[3]}
	} else {
		err =fmt.Errorf("command : rezom [min (begin time mmhh)] [max (end time mmhh)] [ip adrress]")
		}
	return tb, te, ipadr, err
}	
