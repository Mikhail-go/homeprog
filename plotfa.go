package main

import (
	"os"
	"fmt"
	"net/http"
)
import (
	"flag"
	"math"
	"github.com/ajstarks/svgo"
)
import (
	"time"
)
//
	const (
		wplot	= 1100	//ширина графика svg
		maxdp = 300 // количество точек данных на графике
		maxp1 = 7 // максимальная активная мощность по одной фазе 
		imax = 35 // макс. ток фазы (А)
		ufmin = 210 // минимальное фазное напряжение
		ufmax = 240 // и максимальное 
		p3max = 15 // максимальная полная мощность  (квт)
		p1max = 7 // максимальная активная мощность по одной фазе (квт)

	)
const (
	globalfmt = "font-family:%s;font-size:%dpt;stroke-width:%dpx"
	linestyle = "fill:none;stroke:"
	linefmt   = "fill:none;stroke:%s"
	barfmt    = linefmt + ";stroke-width:%dpx"
	ticfmt    = "stroke:rgb(200,200,200);stroke-width:1px"
	labelfmt  = ticfmt + ";text-anchor:end;fill:black"
	textfmt   = "stroke:none;baseline-shift:-33.3%"
	//smallint  = -(1 << 30)
)
	// rawdata defines data as float64 x,y coordinates
type rawdata struct {
	x string	//для временного интервала в формате hhmmss
	y float64
}
type options map[string]bool
type attributes map[string]string
type measures map[string]int

// plotset defines plot metadata
type plotset struct {
	opt  options
	attr attributes
	size measures
}
var (	
	canvas = svg.New(os.Stdout)
	plotopt = options{}
	plotattr = attributes{}
	plotnum = measures{}
	ps = plotset{plotopt, plotattr, plotnum}
	plotw, ploth, plotc, gwidth, gheight, gutter, beginx, beginy int
)
	var dgr [3][5]int  //индексы для svg графика
	//var xstep int	//шаг разметки по x 

// init initializes command flags and sets default options
func init() {

	// boolean options
	showx := flag.Bool("showx", false, "show the xaxis") //true
	showy := flag.Bool("showy", true, "show the yaxis")
	showbar := flag.Bool("showbar", false, "show data bars")
	area := flag.Bool("area", false, "area chart")
	connect := flag.Bool("connect", true, "connect data points")
	showdot := flag.Bool("showdot", false, "show dots")
	showbg := flag.Bool("showbg", true, "show the background color")
	showfile := flag.Bool("showfile", true, "show the filename")
	sameplot := flag.Bool("sameplot", true, "plot on the same frame")

	// attributes
	bgcolor := flag.String("bgcolor", "rgb(240,240,240)", "plot background color")
	barcolor := flag.String("barcolor", "gray", "bar color")
	dotcolor := flag.String("dotcolor", "black", "dot color")
	linecolor := flag.String("linecolor", "gray", "line color")
	areacolor := flag.String("areacolor", "gray", "area color")
	font := flag.String("font", "Calibri,sans", "font")
	labelcolor := flag.String("labelcolor", "black", "label color")
	plotlabel := flag.String("label", "", "plot label")
	//myip := flag.String("ip:", "127.0.0.1", "site ip adress")
	// sizes
	dotsize := flag.Int("dotsize", 2, "dot size")
	linesize := flag.Int("linesize", 1, "line size")
	barsize := flag.Int("barsize", 2, "bar size")
	fontsize := flag.Int("fontsize", 11, "font size")
	xinterval := flag.Int("xint", 6, "x axis interval")	//число точек на интервал (минуты)
	yinterval := flag.Int("yint", 5, "y axis interval") //
	ymin := flag.Int("ymin", 0, "y minimum") //// минимум физ. значения
	ymax := flag.Int("ymax", p3max, "y maximum") // максимум физ.значения (напр. полная мощность)

	// meta options
	flag.IntVar(&beginx, "bx", 100, "initial x")
	flag.IntVar(&beginy, "by", 50, "initial y")
	flag.IntVar(&plotw, "pw", wplot, "plot width")
	flag.IntVar(&ploth, "ph", 600, "plot height")
	flag.IntVar(&plotc, "pc", 2, "plot columns")
	flag.IntVar(&gutter, "gutter", ploth/10, "gutter") //
	flag.IntVar(&gwidth, "width", 1200, "canvas width") //1024
	flag.IntVar(&gheight, "height", 768, "canvas height")

	flag.Parse()

	// fill in the plotset -- all options, attributes, and sizes
	plotopt["showx"] = *showx
	plotopt["showy"] = *showy
	plotopt["showbar"] = *showbar
	plotopt["area"] = *area
	plotopt["connect"] = *connect
	plotopt["showdot"] = *showdot
	plotopt["showbg"] = *showbg
	plotopt["showfile"] = *showfile
	plotopt["sameplot"] = *sameplot

	plotattr["bgcolor"] = *bgcolor
	plotattr["barcolor"] = *barcolor
	plotattr["linecolor"] = *linecolor
	plotattr["dotcolor"] = *dotcolor
	plotattr["areacolor"] = *areacolor
	plotattr["font"] = *font
	plotattr["label"] = *plotlabel
	plotattr["labelcolor"] = *labelcolor
//	plotattr["ip"] = *myip

	plotnum["dotsize"] = *dotsize
	plotnum["linesize"] = *linesize
	plotnum["fontsize"] = *fontsize
	plotnum["xinterval"] = *xinterval
	plotnum["yinterval"] = *yinterval
	plotnum["barsize"] = *barsize
	plotnum["ymin"] = *ymin
	plotnum["ymax"] = *ymax
}
// fmap maps world data to document coordinates
func fmap(value float64, low1 float64, high1 float64, low2 float64, high2 float64) float64 {
	normv := 0.
	if (high1 != low1) { normv = (value-low1)/(high1-low1) }
	return low2 + (high2-low2)*normv
}
// plot places a plot at the specified location with the specified dimemsions
// usinng the specified settings, using the specified data
func plot(x, y, w, h int, settings plotset, d []rawdata) {
	nd := len(d)
	if nd < 2 {
		fmt.Fprintf(os.Stderr, "%d is not enough points to plot\n", len(d))
		return
	}
    minx := secomt(d[0].x) //  начальная точка   сек от старта
	mixb:= minx - float64(int(minx)%60)  // minimum для разметки в минутах
    maxx := secomt(d[nd-1].x) //время конечной точки в сек
	maxl := maxx - float64(int(maxx)%60)  // максимум для разметки в минутах
	//fmt.Println(nd, maxx, maxl)	
	miny := float64(settings.size["ymin"]) //минимум значения i/u/p для графика
	maxy := float64(settings.size["ymax"]) // и максимум
	// Prepare for a area or line chart by allocating
	// polygon coordinates; for the hrizon plot, you need two extra coordinates
	// for the extrema.
	needpoly := settings.opt["area"] || settings.opt["connect"]
	var xpoly, ypoly []int
	if needpoly {
		xpoly = make([]int, nd+2)
		ypoly = make([]int, nd+2)
		// preload the extrema of the polygon,
		// the bottom left and bottom right of the plot's rectangle
		xpoly[0] = x
		ypoly[0] = y + h
		xpoly[nd+1] = x + w
		ypoly[nd+1] = y + h
	}
	// Draw the plot's bounding rectangle
	if settings.opt["showbg"] && !settings.opt["sameplot"] {
		canvas.Rect(x, y, w, h, "fill:"+settings.attr["bgcolor"])
	}
	// Loop through the data, drawing items as specified
	spacer := 10
	canvas.Gstyle(fmt.Sprintf(globalfmt,
		settings.attr["font"], settings.size["fontsize"], settings.size["linesize"]))
//	отрисовка временной шкалы с шагом кратным 60 сек
		imaxl := int(maxl)
		imixb := int(mixb)
		mstep := (imaxl-imixb)/10
		smstep := mstep % 60
	if imaxl >= 60 {
			if smstep != 0 {
			mstep = (mstep - smstep) +60
			}
		for xm := imixb+mstep; xm <= imaxl; xm += mstep {
		xmp := int(fmap(float64(xm), minx, maxx, float64(x), float64(x+w)))		
		//canvas.Text(xmp, (y+h)+(spacer*2), fmt.Sprintf("%v", time.Duration(xm) * time.Second), "text-anchor:middle")
        canvas.Text(xmp, (y+h)+(spacer*2), sprnosec(xm), "text-anchor:middle")
		canvas.Line(xmp, (y + h), xmp, (y+h)+spacer, ticfmt)
		}
	}
    for i, v := range d {
		xp := int(fmap(secomt(v.x), minx, maxx, float64(x), float64(x+w)))
		yp := int(fmap(v.y, miny, maxy, float64(y), float64(y-h)))
		if needpoly {
			xpoly[i+1] = xp
			ypoly[i+1] = yp + h
		}
		if settings.opt["showbar"] {
			canvas.Line(xp, yp+h, xp, y+h,
				fmt.Sprintf(barfmt, settings.attr["barcolor"], settings.size["barsize"]))
		}
		if settings.opt["showdot"] {
			canvas.Circle(xp, yp+h, settings.size["dotsize"], "fill:"+settings.attr["dotcolor"])
		}
		if settings.opt["showx"] {
			if (i+1)%settings.size["xinterval"] == 0 {
		//		//canvas.Text(xp, (y+h)+(spacer*2), fmt.Sprintf("%.1f", v.x/60), "text-anchor:middle")
				canvas.Text(xp, (y+h)+(spacer*2), fmt.Sprintf("%s", v.x), "text-anchor:middle")
				//canvas.Text(xp, (y+h)+(spacer*2), fmt.Sprintf("%s", sdursec(v.x)), "text-anchor:middle")
				canvas.Line(xp, (y + h), xp, (y+h)+spacer, ticfmt)
			}
		}
	}
	//fmt.Println (xpoly, ypoly) //deb
	// Done constructing the points for the area or line plots, display them in one shot
	if settings.opt["area"] {
		canvas.Polygon(xpoly, ypoly, "fill:"+settings.attr["areacolor"])
	}

	if settings.opt["connect"] {
		canvas.Polyline(xpoly[1:nd+1], ypoly[1:nd+1], linestyle+settings.attr["linecolor"])
	}
	// Put on the y axis labels, if specified
	if settings.opt["showy"] {
		bot := math.Floor(miny)
		top := math.Ceil(maxy)
		yrange := top - bot
		interval := yrange / float64(settings.size["yinterval"])
		canvas.Gstyle(labelfmt)
		for yax := bot; yax <= top; yax += interval {
			yaxp := fmap(yax, bot, top, float64(y), float64(y-h))
			canvas.Text(x-spacer, int(yaxp)+h, fmt.Sprintf("%.1f", yax), textfmt)
			canvas.Line(x-spacer, int(yaxp)+h, x, int(yaxp)+h)
		}
		canvas.Gend()
	}
	// Finally, tack on the label, if specified
	if len(settings.attr["label"]) > 0 {
	canvas.Text(x, (y+h)+(spacer*4), fmt.Sprintf("%s",st0) , "font-size:120%;fill:"+settings.attr["labelcolor"])
	canvas.Text(x, (y+h)+(spacer*6), settings.attr["label"], "font-size:120%;fill:"+settings.attr["labelcolor"])
	}

	canvas.Gend()
}
//
func dxyplot3(x, y, i, j int) {
	var ra rawdata
	sstd := std[(dgr[i][j]):]
	ls := len(sstd)
	if ls <= 2 {
	return
	}
	//if ls > maxdp {sstd = sstd[:maxdp]}
	rx := Max.D[i][j]
	for k, _ := range rx {
	if k == 1 {plotattr["linecolor"] = "green" }
	if k == 2 {plotattr["linecolor"] = "blue" }
	data := make([]rawdata, 1)
		for is, td := range (sstd) {
			if is > 0 {
			data = append(data,ra)
			}
		data[is].x = td.t
		dx := omdat(td.d[i][j]) //-			
		data[is].y = valf(i,j,k,dx) //-
		}
	plot(x, y, plotw, ploth, ps, data)
	}
	//dgr[i][j] += maxdp
	//if len(std) <= dgr[i][j] { dgr[i][j] = 0 } 
}

// сервис http://host:8181/pf3
func plotp3(w http.ResponseWriter, req *http.Request) {
	canvas = svg.New(w)
  w.Header().Set("Content-Type", "image/svg+xml")
	canvas.Start(gwidth, gheight)
	canvas.Rect(0, 0, gwidth, gheight, "fill:white") 		
	plotnum["ymax"] = p1max
	plotattr["linecolor"] = "red"
	//plotnum["ymax"] = int(rmax(Max.D[2][0]))+1
	plotattr["label"] = "Активные мощности по фазам (квт, RGB) и реактивная мощность (квар, black) от времени (мин)"
	dxyplot3(beginx, beginy, 2, 0)
	plotattr["linecolor"] = "black"
	dxyplot3(beginx, beginy, 2, 2)
	canvas.End()
}
// сервис http://host:8181/uf3
func plotuf3(w http.ResponseWriter, req *http.Request) {
	canvas = svg.New(w)
  w.Header().Set("Content-Type", "image/svg+xml")
	canvas.Start(gwidth, gheight)
	canvas.Rect(0, 0, gwidth, gheight, "fill:white")
	//plotnum["ymax"] = ufmax
	plotnum["ymax"] = int(rmax(Max.D[1][0]))+1
	plotnum["ymin"] = int(rmin(Min.D[1][0]))
	plotattr["label"] = " Фазные напряжения (RGB)(В) от времени (мин)"
	plotattr["linecolor"] = "red"
	dxyplot3(beginx, beginy, 1,0)
	plotnum["ymin"] = 0 // это минимум для токов и мощностей
	canvas.End()
}
// сервис http://host:8181/if3
func ploti3a(w http.ResponseWriter, req *http.Request) {
	canvas = svg.New(w)
	 w.Header().Set("Content-Type", "image/svg+xml")
	canvas.Start(gwidth, gheight)
	canvas.Rect(0, 0, gwidth, gheight, "fill:white") 		
	plotnum["ymax"] = imax
	plotattr["linecolor"] = "red"
//	plotnum["ymax"] = int(rmax(Max.D[0][0]))+1
	plotattr["label"] = " Токи фаз (А, RGB) и ток обратной последовательности (A, black)  от времени (мин)"
	dxyplot3(beginx, beginy, 0, 0)
	plotattr["linecolor"] = "black"
	dxyplot3(beginx, beginy, 0, 4)
	canvas.End()
}
// сервис http://host:8181/cosfi
func plotcosfi(w http.ResponseWriter, req *http.Request) {
	canvas = svg.New(w)
  w.Header().Set("Content-Type", "image/svg+xml")
	canvas.Start(gwidth, gheight)
	canvas.Rect(0, 0, gwidth, gheight, "fill:white") 		
	plotnum["ymax"] = 1
	//plotnum["ymax"] = int(rmax(Max.D[2][4]))+1
	plotattr["label"] = " Косинус фи по фазам от времени (мин)" 
	plotattr["linecolor"] = "red"
	dxyplot3(beginx, beginy, 2, 4)
	canvas.End()
}
// преобразование в секунды интервала времени hhmmss 
func secomt(t string) float64 {
	tx , _ := time.ParseDuration(t)
	return tx.Seconds()
}
func sprnosec(xm int) string {
     str := fmt.Sprintf("%v", time.Duration(xm) * time.Second)
     lr := len(str)-2 //не нужный хвост 0s удаляем
     return fmt. Sprintf("%s", str[:lr])
}
func rmax(x []float64) float64 {
	rx := 0.
	for _, r := range x {
	if r > rx { rx = r}
	}
	return rx
}
func rmin(x []float64) float64 {
	rx := 65000.
	for _, r := range x {
	if r < rx { rx = r}
	}
	return rx
}
