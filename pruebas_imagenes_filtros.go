package main

import (

	//	"fmt"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	"github.com/zeg/core/trace"
	"github.com/zeg/core/util"

	//	"math"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"gocv.io/x/gocv"
)

/*

	pruebas_imagenes_filtros IMAGEN= PROCESO=

*/
/*

	Las caracteristicas de las placas

	altura placa = 50% ancho placa
	altura caracteres = 40% altura placa
	altura caracteres = 23% ancho todos los caracteres
	ancho de todos los caracteres = 4.2 alto de los caracteres
	ancho de todos los caracteres = 90% ancho placa

		Usare de entrada el estandar NTSC

  NTSC Standard

  0.299 R
  0.587 G
  0.114 B

  La relacion de abajo es para obtener maximo 255
var relacion_brillantez_rojo uint32 = 76
var relacion_brillantez_verde uint32 = 150
var relacion_brillantez_azul uint32 = 29

*/

var relacion_brillantez_rojo uint32 = 76
var relacion_brillantez_verde uint32 = 150
var relacion_brillantez_azul uint32 = 29

var valor_relacion_brillantez uint32 = 256 // 65535

/*

	Arreglo para almacenar el brillo acumulado por cada pixel

*/

var bitmap [][]int

type PuntoStruct struct {
	x      int
	y      int
	brillo uint32
}

var punto_max PuntoStruct

//var global_dx int
//var global_dy int
//var global_angulo float64

const cuantos_angulos = 1800 // Para hacer las decimas de grado

var cos_angulo []float64
var sen_angulo []float64

var porc_cal_gris uint32

func main() {

	var dato string
	var id_seguimiento string
	var err error
	var nombre_imagen string
	var nombre_imagen_2 string
	var ok bool
	var opciones_jpeg jpeg.Options
	var imagen_origen image.Image
	var imagen_prueba image.Image
	//	var imagen_destino image.Image
	//var imagen_hough *image.RGBA
	//	var imagen_destino_rgba image.RGBA
	var imagen_origen_rgba *image.RGBA
	var imagen_prueba_rgba *image.RGBA
	//	var imagen_hough_rgba image.RGBA
	var imagen_salida draw.Image
	var archivo_destino *os.File
	var archivo_origen *os.File
	var archivo_mascara *os.File
	var proceso int64
	var num_parametros int
	var nombre string
	var angulo int
	var nombre_archivo string

	var fconv_grados_radianes float64 = math.Pi / cuantos_angulos
	var indice int
	var direccion int

	var imagen_cv gocv.Mat

	nombre_programa := path.Base(os.Args[0]) // Nombre del programa ejecutado
	id_seguimiento = nombre_programa
	ok = false
	opciones_jpeg.Quality = 100
	proceso = 0
	num_parametros = 0
	angulo = 0

	imagen_cv = gocv.NewMat()
	defer imagen_cv.Close()

	trace.Salida(id_seguimiento, 0, "Iniciando el programa para prueba de filtros de imagenes >", nombre_programa, "<")
	trace.Salida(id_seguimiento, 0, "Version 1.0")

	cos_angulo = make([]float64, cuantos_angulos)
	for indice = 0; indice < cuantos_angulos; indice++ {
		cos_angulo[indice] = math.Cos(float64(indice) * fconv_grados_radianes) // Cos lo obtiene en radianes por eso se convierte
	}

	sen_angulo = make([]float64, cuantos_angulos)
	for indice = 0; indice < cuantos_angulos; indice++ {
		sen_angulo[indice] = math.Sin(float64(indice) * fconv_grados_radianes) // Sin lo obtiene en radianes por eso se convierte
	}

	porc_cal_gris = 5 // Porcentaje de desviacion en colores para calificar como gris

	/*

		Busco los parametros en linea de comandos

	*/

	for indice, _ := range os.Args {

		var parametro string

		donde := strings.Index(os.Args[indice], "=")

		if donde != -1 {
			parametro = os.Args[indice][:donde]
			if donde < len(os.Args[indice]) {
				dato = os.Args[indice][donde+1:]
			} else {
				dato = ""
			}
		} else {
			parametro = os.Args[indice]
		}

		if (parametro == "?") || (strings.ToLower(parametro) == "help") { // OJO verificar el metodo correcto
			trace.Salida(id_seguimiento, 0, "Se usa escribiendo:")
			trace.Salida(id_seguimiento, 0, "	", os.Args[0], " PAR1=AAAA PAR2=BBBB ...")
			trace.Salida(id_seguimiento, 0, "	", os.Args[0], " IMAGEN=xxx PROCESO=n IMAGEN2=xxx ANGULO=n.n NIVEL_TRACE=n")
			trace.Salida(id_seguimiento, 0, "	ANGULO para las rotaciones el angulo puede llevar decimales, ejemplo 38.4, el signo es importante 1 es contrareloj")
			trace.Salida(id_seguimiento, 0, "	Si no se especifica el nivel de trace por omision lo pone en 0")
			trace.Salida(id_seguimiento, 0, "	Ejemplo.- escribir '", os.Args[0], "' es lo mismo que poner '", os.Args[0], " NIVEL_TRACE=0'")
			break
		}

		if parametro == "NIVEL_TRACE" {
			var valor int64

			valor, err = strconv.ParseInt(dato, 10, 0)

			if err == nil {
				trace.CambiarNivelTrace(valor)
				if trace.ObtenNivelTrace() < 0 {
					trace.CambiarNivelTrace(0)
				}
				if trace.ObtenNivelTrace() > 30 {
					trace.CambiarNivelTrace(30)
				}

				if trace.ObtenNivelTrace() > 0 {
					trace.Salida("0", 5, "NIVEL_TRACE >", trace.ObtenNivelTrace(), "<")
				}
			}
		}

		if parametro == "IMAGEN" {
			nombre_imagen = dato
			num_parametros++
		}

		if parametro == "IMAGEN2" {
			nombre_imagen_2 = dato
			num_parametros++
		}

		if parametro == "PROCESO" {
			proceso, err = strconv.ParseInt(dato, 10, 0)
			if err == nil {
				num_parametros++
			}
		}

		if parametro == "ANGULO" {
			angulo_obt, err := strconv.ParseFloat(dato, 64)
			if err == nil {
				if angulo_obt < 0 {
					direccion = -1
				} else {
					direccion = 1
				}
				angulo = direccion * int(angulo_obt*10)
			}
		}

	}

	if num_parametros >= 2 {
		ok = true
	}

	/*

		Para la imagen de entrada

	*/

	if ok {
		archivo_origen, err = os.Open(nombre_imagen)

		if err != nil {
			texto := "Error" + err.Error() + "al abrir" + nombre_imagen
			err = errors.New(texto)
		} else {
			imagen_origen, err = jpeg.Decode(archivo_origen)
		}

		if err == nil {
			if img, ok := imagen_origen.(*image.RGBA); !ok {

				b := imagen_origen.Bounds()
				imagen_origen_rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
				draw.Draw(imagen_origen_rgba, imagen_origen_rgba.Bounds(), imagen_origen, b.Min, draw.Src)

			} else {
				imagen_origen_rgba = img
			}
		}

		if err == nil {
			err = archivo_origen.Close()
		}
	}

	if err == nil {
		if proceso == 15 {
			archivo_mascara, err = os.Open(nombre_imagen_2)

			if err == nil {
				imagen_prueba, err = jpeg.Decode(archivo_mascara)
			}

			if err == nil {
				if img, ok := imagen_prueba.(*image.RGBA); !ok {

					b := imagen_prueba.Bounds()
					imagen_prueba_rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
					draw.Draw(imagen_prueba_rgba, imagen_prueba_rgba.Bounds(), imagen_prueba, b.Min, draw.Src)

				} else {

					imagen_prueba_rgba = img

				}
			}

			if err == nil {
				err = archivo_mascara.Close()
			}
		}
	}

	if err == nil {
		/*

			Para la imagen de salida

		*/

		nombre = ""
		switch proceso {
		case 1:
			nombre = "_rotada"
		case 2:
			nombre = "_scolor"
		case 3:
			nombre = "_picosv"
		case 4:
			nombre = "_picosh"
		case 5:
			nombre = "_completada"
		case 6:
			nombre = "_nsobel"
		case 7:
			nombre = "_sobel_bin"
		case 8:
			nombre = "_paso2"
		case 9:
			nombre = ""
		case 10:
			nombre = "_gris"
		case 11:
			nombre = "_sup"
		case 12:
			nombre = "_lineasblancasv"
		case 13:
			nombre = "_lineasblancash"
		case 14:
			nombre = "_marco"
		case 15:
			nombre = ""
		case 16:
			nombre = "_prev_hugh"
		case 17:
			nombre = "_sobelh"
		case 18:
			nombre = "_sobelv"
		}

		if nombre != "" {

			if err == nil {
				lugar1 := util.RetroIndex(nombre_imagen, "/")
				lugar2 := util.RetroIndex(nombre_imagen, ".")

				if lugar1 != -1 {
					if lugar2 != -1 {
						nombre_archivo = nombre_imagen[lugar1+1 : lugar2]
					} else {
						nombre_archivo = nombre_imagen[lugar1+1:]
					}
				} else {
					if lugar2 != -1 {
						nombre_archivo = nombre_imagen[:lugar2]
					} else {
						nombre_archivo = nombre_imagen
					}
				}
			}

			if err == nil {
				nombre_salida := nombre_archivo + nombre + ".jpg"
				archivo_destino, err = os.Create(nombre_salida)
			}
		}

		if err == nil {
			//imagen_salida = CrearImagenColor(172, 58)

			/*

				Filtros

			*/

			switch proceso {
			case 1:
				imagen_salida = RotarImagen(*imagen_origen_rgba, angulo, direccion)
			case 2:
				var borde_inicial uint8 = 0
				var borde_final uint8 = 80
				imagen_salida = EliminaColores(*imagen_origen_rgba, borde_inicial, borde_final)
			case 3:
				imagen_salida, _, _ = GenerarPicosVallesV(*imagen_origen_rgba)
			case 4:
				imagen_salida = GenerarPicosVallesH(*imagen_origen_rgba, 28, 67) // Valores que salen de GenerarPicosVallesV 24, 76
			case 5:
				imagen_salida = Completar(*imagen_origen_rgba)
			case 6:
				imagen_salida = NuevoFiltroSobel(*imagen_origen_rgba)
			case 7:
				imagen_salida = SobelNegrosBin(*imagen_origen_rgba) // Proceso para preparar la imagen para hugh
			case 8:
				imagen_salida = PreparaPaso2(*imagen_origen_rgba, 10)
			case 9:
				Enfoque(*imagen_origen_rgba)
			case 10:
				imagen_salida = ConvierteRGBaGris(*imagen_origen_rgba)
			case 11:
				imagen_salida = SuperficiesRellenas(*imagen_origen_rgba)
			case 12:
				imagen_salida = LineasBlancasV(*imagen_origen_rgba)
			case 13:
				imagen_salida = LineasBlancasH(*imagen_origen_rgba)
			case 14:
				imagen_salida = Marco(*imagen_origen_rgba, 10)
			case 15:
				// t_inicio := time.Now()
				valor := DiferentesMascara(imagen_origen, imagen_prueba)
				// t_final := time.Now()
				// trace.Salida(id_seguimiento, 0, "Vehiculo ", t_final.Format("20060102150405.000000"), t_inicio.Format("20060102150405.000000"))
				trace.Salida(id_seguimiento, 0, "	valor >", valor, "<")
			case 17:
				imagen_salida = SobelH(*imagen_origen_rgba)
			case 18:
				imagen_salida = SobelV(*imagen_origen_rgba)
			}

		}

		if nombre != "" {
			if err == nil {
				err = jpeg.Encode(archivo_destino, imagen_salida, &opciones_jpeg)
			}

			if err == nil {
				err = archivo_destino.Close()
			}
		}

	} // Fin del ok de todos los parametros

	if err != nil {
		trace.Salida(id_seguimiento, 0, "	Error >", err.Error(), "<")
	}

	trace.Salida(id_seguimiento, 0, "Terminando ", nombre_programa)

} // Termina main

type Rectangulo struct {
	rect_x0 float64
	rect_y0 float64
	rect_x1 float64
	rect_y1 float64
}

type Punto struct {
	x float64
	y float64
}

/*
Busca el valor de diferencia entre dos imagenes
la imagen a probar debe de ser de mismo tamanio de la mascara
basado mas o menos en Otsuka–Ochiai coefficient
*/
func Diferentes(imagen_mascara image.Image, imagen_prueba image.Image) (resultado float64) {

	var rectangulo Rectangulo
	var punto Punto
	var indice int
	var valor int
	var valor_f float64
	var valor_medio float64
	var histograma_mascara []int
	var histograma_prueba []int
	var alfa float64
	var beta float64
	var ancho_imagen float64
	var alto_imagen float64

	rectangulo.rect_x0 = 0
	rectangulo.rect_y0 = 0
	rectangulo.rect_x1 = 0.140625    // 180
	rectangulo.rect_y1 = 0.104166667 // 75

	min_x := float64(imagen_mascara.Bounds().Min.X)
	min_y := float64(imagen_mascara.Bounds().Min.Y)
	max_x := float64(imagen_mascara.Bounds().Max.X)
	max_y := float64(imagen_mascara.Bounds().Max.Y)

	punto.x = 0.4921875          // 630
	punto.y = 0.4444444444444444 // 320

	ancho_imagen = max_x - min_x
	alto_imagen = max_y - min_y

	punto_extraccion := image.Point{int(punto.x * ancho_imagen), int(punto.y * alto_imagen)}

	rectangulo_imagen := image.Rect(int(rectangulo.rect_x0*ancho_imagen), int(rectangulo.rect_y0*alto_imagen), int(rectangulo.rect_x1*ancho_imagen), int(rectangulo.rect_y1*alto_imagen))
	mascara := image.NewRGBA(rectangulo_imagen)
	draw.Draw(mascara, mascara.Bounds(), imagen_mascara, punto_extraccion, draw.Src)
	histograma_mascara = CreaHistogramaGrises(*mascara)

	prueba := image.NewRGBA(rectangulo_imagen)
	draw.Draw(prueba, prueba.Bounds(), imagen_prueba, punto_extraccion, draw.Src)
	histograma_prueba = CreaHistogramaGrises(*prueba)

	valor = 0
	for indice = 0; indice < 256; indice++ {
		valor += (indice + 1) * histograma_mascara[indice]
	}
	alfa = float64(valor) / 256

	valor = 0
	for indice = 0; indice < 256; indice++ {
		valor += (indice + 1) * histograma_prueba[indice]
	}
	beta = float64(valor) / 256

	valor_medio = math.Sqrt(alfa * beta)

	valor_f = 0
	for indice = 0; indice < 256; indice++ {
		valor_f += math.Abs((float64(indice) + 1) * (float64(histograma_mascara[indice]) - float64(histograma_prueba[indice])))
	}
	valor_f = valor_f / 256

	resultado = valor_f / valor_medio

	return resultado
}

/*
Busca el valor de diferencia entre dos imagenes
la imagen a probar debe de ser de mismo tamanio de la mascara
basado mas o menos en Otsuka–Ochiai coefficient
*/
func DiferentesMascara(imagen_mascara image.Image, imagen_prueba image.Image) (resultado float64) {

	var rectangulo Rectangulo
	var punto Punto
	var indice int
	var valor_f float64
	var valor_medio float64
	var histograma_mascara []int
	var histograma_prueba []int
	var alfa float64
	var beta float64

	rectangulo.rect_x0 = 0
	rectangulo.rect_y0 = 0
	rectangulo.rect_x1 = 0.140625    // 180
	rectangulo.rect_y1 = 0.104166667 // 75

	punto.x = 0.4921875          // 630
	punto.y = 0.4444444444444444 // 320

	alfa, histograma_mascara = ExtraeConstanteHistograma(imagen_mascara, rectangulo, punto)
	beta, histograma_prueba = ExtraeConstanteHistograma(imagen_prueba, rectangulo, punto)

	valor_medio = math.Sqrt(alfa * beta)

	valor_f = 0
	for indice = 0; indice < 256; indice++ {
		valor_f += math.Abs((float64(indice) + 1) * (float64(histograma_mascara[indice]) - float64(histograma_prueba[indice])))
	}
	valor_f = valor_f / 256

	resultado = valor_f / valor_medio

	return resultado
}

/*
Extrae el histograma y la constante de la subimagen
*/
func ExtraeConstanteHistograma(imagen image.Image, rectangulo Rectangulo, punto Punto) (constante float64, histograma []int) {

	var indice int
	var valor int
	var ancho_imagen float64
	var alto_imagen float64

	min_x := float64(imagen.Bounds().Min.X)
	min_y := float64(imagen.Bounds().Min.Y)
	max_x := float64(imagen.Bounds().Max.X)
	max_y := float64(imagen.Bounds().Max.Y)

	ancho_imagen = max_x - min_x
	alto_imagen = max_y - min_y

	punto_extraccion := image.Point{int(punto.x * ancho_imagen), int(punto.y * alto_imagen)}

	rectangulo_imagen := image.Rect(int(rectangulo.rect_x0*ancho_imagen), int(rectangulo.rect_y0*alto_imagen), int(rectangulo.rect_x1*ancho_imagen), int(rectangulo.rect_y1*alto_imagen))
	mascara := image.NewRGBA(rectangulo_imagen)
	draw.Draw(mascara, mascara.Bounds(), imagen, punto_extraccion, draw.Src)
	histograma = CreaHistogramaGrises(*mascara)

	valor = 0
	for indice = 0; indice < 256; indice++ {
		valor += (indice + 1) * histograma[indice]
	}
	constante = float64(valor) / 256

	return constante, histograma
}

/*
Busca el valor de diferencia entre dos imagenes
la imagen a probar debe de ser de mismo tamanio de la mascara
*/
func Diferentes1(imagen_mascara image.Image, imagen_prueba image.Image) (resultado float64) {

	var min_x float64
	var min_y float64
	var max_x float64
	var max_y float64
	var indice int
	var valor uint
	var valor_f float64
	var suma_brillo_a float64
	var suma_brillo_b float64
	var valor_medio float64
	var histograma_mascara []int
	var histograma_prueba []int
	var opciones_jpeg jpeg.Options

	opciones_jpeg.Quality = 100

	trace.Salida("Diferentes", 0, "Entro")

	min_x = 0
	min_y = 0
	max_x = 0.140625    // 180
	max_y = 0.104166667 // 75

	rectangulo := image.Rect(int(min_x), int(min_y), int(max_x*1280), int(max_y*720))
	mascara := image.NewRGBA(rectangulo)
	draw.Draw(mascara, mascara.Bounds(), imagen_mascara, image.Point{630, 320}, draw.Src)
	histograma_mascara = CreaHistogramaGrises(*mascara)

	trace.Salida("Diferentes", 0, "histograma_mascara >", histograma_mascara, "<")

	prueba := image.NewRGBA(rectangulo)
	draw.Draw(prueba, prueba.Bounds(), imagen_prueba, image.Point{630, 320}, draw.Src)
	histograma_prueba = CreaHistogramaGrises(*prueba)

	trace.Salida("Diferentes", 0, "histograma_prueba >", histograma_prueba, "<")

	archivo_destino, err := os.Create("sal.jpg")
	if err != nil {
		trace.Salida("Diferentes", 0, "err >", err.Error())
	} else {
		trace.Salida("Diferentes", 0, "sin error")
	}
	err = jpeg.Encode(archivo_destino, prueba, &opciones_jpeg)
	if err != nil {
		trace.Salida("Diferentes", 0, "err >", err.Error())
	} else {
		trace.Salida("Diferentes", 0, "sin error")
	}

	valor = 0
	for indice = 0; indice < 256; indice++ {
		valor += uint(histograma_mascara[indice] * indice)
	}
	suma_brillo_a = float64(valor) / 256

	trace.Salida("Diferentes", 0, "suma_brillo_a >", suma_brillo_a, "<")

	valor = 0
	for indice = 0; indice < 256; indice++ {
		valor += uint(histograma_prueba[indice] * indice)
	}
	suma_brillo_b = float64(valor) / 256

	trace.Salida("Diferentes", 0, "suma_brillo_b >", suma_brillo_b, "<")

	valor_medio = math.Sqrt(suma_brillo_a * suma_brillo_b)

	trace.Salida("Diferentes", 0, "valor_medio >", valor_medio, "<")

	valor_f = 0
	for indice = 0; indice < 256; indice++ {
		valor_f += math.Abs(float64(histograma_mascara[indice]) - float64(histograma_prueba[indice]))

		trace.Salida("Diferentes", 0, "temp >", math.Abs(float64(histograma_mascara[indice])-float64(histograma_prueba[indice]))/valor_medio, "<")

	}

	resultado = valor_f / valor_medio

	return resultado
}

/*
Crea un histograma de grises entre 0 y 255
*/
func CreaHistogramaGrises(imagen_entrada image.RGBA) (histograma []int) {

	var indice_x int
	var indice_y int
	var valor uint

	histograma = make([]int, 256)

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	width := max_x - min_x
	height := max_y - min_y

	for indice_x = 0; indice_x < width; indice_x++ {
		for indice_y = 0; indice_y < height; indice_y++ {
			valor = ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y)
			histograma[valor]++
		}
	}

	return histograma
}

/*
Filtro numero 11
Busca bordes y rellenos contra un limite
*/
func SuperficiesRellenas(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	//	var rojo_global, verde_global, azul_global, brillo_global float64
	var valor, valor2 int
	var brillo uint8
	var brillo_central uint8
	var imagen_grises draw.Image

	var brillantez []int64
	//	var distribucion_brillo []int64
	var cuantos_puntos int64
	var color_calculado uint8

	var distribucion_brillo []uint8
	distribucion_brillo = make([]uint8, 10)

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	brillantez = make([]int64, 256)
	//	distribucion_brillo = make([]int64, 10)
	cuantos_puntos = int64((max_y - min_y) * (max_x - min_x))

	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)
	imagen_grises = CrearImagenColor(max_x-min_x, max_y-min_y)

	// Crea imagen en grises
	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			brillo = uint8(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y))
			brillantez[int(brillo)]++

			color_calculado := brillo
			imagen_grises.Set(indice_x, indice_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})
		}
	}

	// Encuentra el 10%
	var contador int64
	var decimos int64 = 1
	var indice uint8
	for indice = 0; indice < 255; indice++ {
		contador += brillantez[indice]
		if contador >= (cuantos_puntos*decimos)/10 {
			decimos = int64(contador / (cuantos_puntos / 10))
			distribucion_brillo[decimos-1] = indice - 1
			decimos++
		}
	}

	// Busca bordes y no bordes
	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			brillo_central = uint8(ImagenBrillantezGris(imagen_grises, indice_x, indice_y))

			/*

			  Matriz a trabajar

			   1  2  1
			   0  X  0
			  -1 -2 -1

			*/

			valor = int(ImagenBrillantezGris(imagen_grises, indice_x-1, indice_y-1))
			valor += 2 * int(ImagenBrillantezGris(imagen_grises, indice_x, indice_y-1))
			valor += int(ImagenBrillantezGris(imagen_grises, indice_x+1, indice_y-1))
			valor -= int(ImagenBrillantezGris(imagen_grises, indice_x-1, indice_y+1))
			valor -= 2 * int(ImagenBrillantezGris(imagen_grises, indice_x, indice_y+1))
			valor -= int(ImagenBrillantezGris(imagen_grises, indice_x+1, indice_y+1))

			if valor < 0 {
				valor = -1 * valor
			}

			/*
			   Matriz a trabajar

			   -1 0 1
			   -2 X 2
			   -1 0 1

			*/

			valor2 = int(ImagenBrillantezGris(imagen_grises, indice_x+1, indice_y-1))
			valor2 += 2 * int(ImagenBrillantezGris(imagen_grises, indice_x+1, indice_y))
			valor2 += int(ImagenBrillantezGris(imagen_grises, indice_x+1, indice_y+1))
			valor2 -= int(ImagenBrillantezGris(imagen_grises, indice_x-1, indice_y-1))
			valor2 -= 2 * int(ImagenBrillantezGris(imagen_grises, indice_x-1, indice_y))
			valor2 -= int(ImagenBrillantezGris(imagen_grises, indice_x-1, indice_y+1))

			if valor2 < 0 {
				valor2 = -1 * valor2
			}

			if indice_x == 51 && indice_y == 39 { // 39
				trace.Salida("SuperficiesRellenas", 0, "  interno valor >", valor, "< valor2 >", valor2, "<")
			}

			if indice_x == 50 && indice_y == 37 { // 132
				trace.Salida("SuperficiesRellenas", 0, "  borde valor 37 >", valor, "< valor2 >", valor2, "<")
			}

			if indice_x == 50 && indice_y == 38 { // 63
				trace.Salida("SuperficiesRellenas", 0, "  borde valor 38 >", valor, "< valor2 >", valor2, "<")
			}

			valor += valor2
			valor /= int(brillo_central)

			if brillo_central < distribucion_brillo[3] { // Es negro
				color_calculado = 0
			} else {
				if brillo_central > distribucion_brillo[4] { // Es blanco
					color_calculado = 255
				} else {
					if valor < 3 { // Es una planicie blanca
						color_calculado = 255
					} else { // Es un borde
						color_calculado = 0
					}
				}
			}

			imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})
		}
	}

	return imagen_sin_colores
}

/*
Filtro numero 12
Busca las lineas blancas mas grandes por cada punto vertical
La imagen de entrada esta en grises de 0 a 255
*/
func LineasBlancasV(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	var lineas []int64

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	lineas = make([]int64, max_y-min_y)

	imagen_sin_colores = CrearImagenCualquierColor(max_x-min_x, max_y-min_y, color.Black)

	var linea_blanca int64
	var linea_mas_grande int64
	for indice_y = min_y; indice_y < max_y; indice_y++ {
		linea_mas_grande = 0
		linea_blanca = 0
		for indice_x = min_x; indice_x < max_x; indice_x++ {
			if ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y) > 127 {
				linea_blanca++
				if linea_blanca >= linea_mas_grande {
					linea_mas_grande = linea_blanca
				}
			} else {
				linea_blanca = 0
			}
		}
		lineas[indice_y-min_y] = linea_mas_grande
	}

	for indice_y = 0; indice_y < max_y-min_y; indice_y++ {
		if lineas[indice_y] > 0 {
			LineaRgbaH(imagen_sin_colores, 0, indice_y, int(lineas[indice_y]), color.RGBA{255, 255, 255, 255})
		}
	}

	return imagen_sin_colores
}

/*
Filtro numero 13
Busca las lineas blancas mas grandes por cada punto horizontal
La imagen de entrada esta en grises de 0 a 255
*/
func LineasBlancasH(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	var lineas []int64

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	lineas = make([]int64, max_x-min_x)

	imagen_sin_colores = CrearImagenCualquierColor(max_x-min_x, max_y-min_y, color.Black)

	var linea_blanca int64
	var linea_mas_grande int64
	for indice_x = min_x; indice_x < max_x; indice_x++ {
		linea_mas_grande = 0
		linea_blanca = 0
		for indice_y = min_y; indice_y < max_y; indice_y++ {
			if ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y) > 127 {
				linea_blanca++
				if linea_blanca >= linea_mas_grande {
					linea_mas_grande = linea_blanca
				}
			} else {
				linea_blanca = 0
			}
		}
		lineas[indice_x-min_x] = linea_mas_grande
	}

	for indice_x = 0; indice_x < max_x-min_x; indice_x++ {
		if lineas[indice_x] > 0 {
			LineaRgbaV(imagen_sin_colores, indice_x, 0, int(lineas[indice_x]), color.RGBA{255, 255, 255, 255})
		}
	}

	return imagen_sin_colores
}

/*
Calcular la posible falta de enfoque con un laplaciano
*/
func EnfoqueLaplaciano(imagen_entrada image.RGBA) {

	var indice_x int
	var indice_y int
	var rojo_global, verde_global, azul_global, brillo_global float64

	// Intentemos por color
	var rojo_acumulado, verde_acumulado, azul_acumulado uint32
	var rojo_inicial, verde_inicial, azul_inicial uint32
	var rojo_temp, verde_temp, azul_temp uint32

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			/*

			   Matriz a trabajar

			    0   1  0
			    1  -4  1
			    0   1  0

			*/

			rojo_acumulado, verde_acumulado, azul_acumulado, _ = imagen_entrada.At(indice_x, indice_y).RGBA()
			rojo_inicial = rojo_acumulado
			verde_inicial = verde_acumulado
			azul_inicial = azul_acumulado
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x-1, indice_y).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x, indice_y+1).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x+1, indice_y).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x, indice_y-1).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp

			// Los colores global van desde 0 hasta 5

			if rojo_inicial > 0 {
				rojo_global += float64(rojo_acumulado) / float64(rojo_inicial)
			}
			if verde_inicial > 0 {
				verde_global += float64(verde_acumulado) / float64(verde_inicial)
			}
			if azul_inicial > 0 {
				azul_global += float64(azul_acumulado) / float64(azul_inicial)
			}

			//if (indice_x == 87) && (indice_y == 41) {
			//if float64(azul_acumulado)/float64(azul_inicial) > 5 {
			//	trace.Salida("Enfoque", 0, "  azul_acumulado >", azul_acumulado, "< azul_inicial >", azul_inicial, "< indice_x >", indice_x, "< indice_y >", indice_y, "<")
			//}

		}
	}

	brillo_global = ((rojo_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_rojo)/256 + (verde_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_verde)/256 + (azul_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_azul)/256)
	trace.Salida("Enfoque", 0, "  rojo_acumulado <", rojo_global/float64((max_x-min_x)*(max_y-min_y)), "< verde_acumulado >", verde_global/float64((max_x-min_x)*(max_y-min_y)), "< azul_acumulado >", azul_global/float64((max_x-min_x)*(max_y-min_y)), "< brillo >", brillo_global, "<")
}

/*
Convierte imagen RGB a gris
*/
func ConvierteRGBaGris(imagen_entrada image.RGBA) (imagen_gris draw.Image) {

	var indice_x int
	var indice_y int
	var brillo uint8

	var brillantez []int64
	var distribucion_brillo []int64
	var cuantos_puntos int64

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	brillantez = make([]int64, 256)
	distribucion_brillo = make([]int64, 10)
	cuantos_puntos = int64((max_y - min_y) * (max_x - min_x))

	fmt.Printf("  cuantos_puntos %d\n", cuantos_puntos)

	imagen_gris = CrearImagenColor(max_x-min_x, max_y-min_y)

	var maximo_puntos_gris int64 = 0

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			brillo = uint8(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y))
			brillantez[int(brillo)]++

			imagen_gris.Set(indice_x, indice_y, color.RGBA{brillo, brillo, brillo, 255})
		}
	}

	var mediana int64

	var contador int64
	var decimos int64 = 1
	var indice int64
	for indice = 0; indice < 256; indice++ {

		if brillantez[indice] >= brillantez[maximo_puntos_gris] {
			maximo_puntos_gris = indice
		}

		contador += brillantez[indice]
		if contador >= (cuantos_puntos*decimos)/10 {
			decimos = int64(contador / (cuantos_puntos / 10))

			fmt.Printf("  decimos %d, indice %d, contador %d\n", decimos, indice, contador)
			distribucion_brillo[decimos-1] = indice - 1
			decimos++
		}

		if contador < cuantos_puntos/2 {
			mediana = indice
		}
	}

	fmt.Printf("  maximo_puntos_gris %d indice %d\n", brillantez[maximo_puntos_gris], maximo_puntos_gris)
	fmt.Printf("  mediana %d\n", mediana)

	for indice = 0; indice < 256; indice++ {
		fmt.Printf("%d ", brillantez[indice])
	}

	for indice = 0; indice < 10; indice++ {
		trace.Salida("ConvierteRGBaGris", 0, "  indice >", indice, "< distribucion_brillo[indice] >", distribucion_brillo[indice], "<")
	}

	return imagen_gris
}

/*
Convierte una imagen de go a MAT de OpenCV
*/
func ConvertirImageToMat(imagen_entrada image.Image) (mat gocv.Mat, err error) {

	var indice_x int
	var indice_y int

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y
	x := imagen_entrada.Bounds().Dx()
	y := imagen_entrada.Bounds().Dy()

	bytes := make([]byte, 0, x*y)

	for indice_x = min_x + 1; indice_x < max_x+1; indice_x++ {
		for indice_y = min_y + 1; indice_y < max_y+1; indice_y++ {
			r, g, b, a := imagen_entrada.At(indice_x, indice_y).RGBA()
			bytes = append(bytes, byte(b>>8))
			bytes = append(bytes, byte(g>>8))
			bytes = append(bytes, byte(r>>8))
			bytes = append(bytes, byte(a>>8))
		}
	}

	mat, err = gocv.NewMatFromBytes(y, x, gocv.MatTypeCV8UC4, bytes)

	return mat, err
}

/*
Calcular la posible falta de enfoque
*/
func Enfoque(imagen_entrada image.RGBA) {

	var indice_x int
	var indice_y int
	var rojo_global, verde_global, azul_global, brillo_global float64

	// Intentemos por color
	var rojo_acumulado, verde_acumulado, azul_acumulado uint32
	var rojo_inicial, verde_inicial, azul_inicial uint32
	var rojo_temp, verde_temp, azul_temp uint32

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			/*

			   Matriz a trabajar

			    0  1  0
			    1  1  1
			    0  1  0

			*/

			rojo_acumulado, verde_acumulado, azul_acumulado, _ = imagen_entrada.At(indice_x, indice_y).RGBA()
			rojo_inicial = rojo_acumulado
			verde_inicial = verde_acumulado
			azul_inicial = azul_acumulado
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x-1, indice_y).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x, indice_y+1).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x+1, indice_y).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp
			rojo_temp, verde_temp, azul_temp, _ = imagen_entrada.At(indice_x, indice_y-1).RGBA()
			rojo_acumulado += rojo_temp
			verde_acumulado += verde_temp
			azul_acumulado += azul_temp

			// Los colores global van desde 0 hasta 5

			if rojo_inicial > 0 {
				rojo_global += float64(rojo_acumulado) / float64(rojo_inicial)
			}
			if verde_inicial > 0 {
				verde_global += float64(verde_acumulado) / float64(verde_inicial)
			}
			if azul_inicial > 0 {
				azul_global += float64(azul_acumulado) / float64(azul_inicial)
			}

			//if (indice_x == 87) && (indice_y == 41) {
			//if float64(azul_acumulado)/float64(azul_inicial) > 5 {
			//	trace.Salida("Enfoque", 0, "  azul_acumulado >", azul_acumulado, "< azul_inicial >", azul_inicial, "< indice_x >", indice_x, "< indice_y >", indice_y, "<")
			//}

		}
	}

	brillo_global = ((rojo_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_rojo)/256 + (verde_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_verde)/256 + (azul_global/float64((max_x-min_x)*(max_y-min_y)))*float64(relacion_brillantez_azul)/256)
	trace.Salida("Enfoque", 0, "  rojo_acumulado <", rojo_global/float64((max_x-min_x)*(max_y-min_y)), "< verde_acumulado >", verde_global/float64((max_x-min_x)*(max_y-min_y)), "< azul_acumulado >", azul_global/float64((max_x-min_x)*(max_y-min_y)), "< brillo >", brillo_global, "<")
}

/*
Filtro para completar, basado en erosion y dilatacion
*/
func Completar(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	var valor int

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)

	for indice_x = min_x; indice_x < max_x+1; indice_x++ {
		for indice_y = min_y; indice_y < max_y+1; indice_y++ {
			/*
				if (indice_x == 18) && (indice_y == 29) {
					trace.Salida("Completar", 0, "  brillo <", int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y)), "< indice_x >", indice_x, "< indice_y >", indice_y, "<")
				}
			*/
			/*

			   Matriz a trabajar

			    1  1  1
			    1  X  1
			    1  1  1

			*/

			valor = 0
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y-1)) > 128 {
				valor = 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y-1)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y-1)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y+1)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y+1)) > 128 {
				valor += 1
			}
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y+1)) > 128 {
				valor += 1
			}
			/*
				if (indice_x == 18) && (indice_y == 29) {
					trace.Salida("Completar", 0, "  valor <", valor, "<")
				}
			*/
			if int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y)) > 128 {
				if valor <= 3 { // Maximo 3 pixeles brillantez y lo convierto a negro
					imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{0, 0, 0, 255})
				} else {
					imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{255, 255, 255, 255})
				}
			} else {
				if valor >= 7 { // Mas de 7 pixeles convierto a blanco
					imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{255, 255, 255, 255})
				} else {
					imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{0, 0, 0, 255})
				}

			}

		}
	}

	return imagen_sin_colores
}

/*
Elimina los colores diferentes a grises
*/
func EliminaColores(imagen_entrada image.RGBA, borde_inicial uint8, borde_final uint8) (imagen_gris draw.Image) {

	var indice_x int
	var indice_y int
	var es_gris bool
	var brillantez int
	var brillo uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_gris = CrearImagenColor(max_x-min_x, max_y-min_y)

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {
			es_gris, brillantez = BrilloEsGris(imagen_entrada, indice_x, indice_y)

			brillo = uint8(brillantez) // CGC debo usar los bits altos

			if es_gris {
				if (brillo >= borde_inicial) && (brillo <= borde_final) {
					imagen_gris.Set(indice_x, indice_y, color.RGBA{brillo, brillo, brillo, 255})
				}
			}

		}
	}

	return imagen_gris
}

/*
Cuando el punto que le envian es algún tono de gris devuelve true y el brillo,
caso contrario devuelve falso y sin brillo
*/
func BrilloEsGris(imagen_entrada image.RGBA, entrada_x int, entrada_y int) (es_gris bool, brillo int) {

	var delta uint32
	var valor uint32

	delta = (256 * porc_cal_gris) / 100

	rojo, verde, azul, _ := imagen_entrada.At(entrada_x, entrada_y).RGBA()

	rojo = rojo >> 8
	verde = verde >> 8
	azul = azul >> 8

	// trace.Salida("BrilloEsGris", 0, "  rojo <", rojo, "< verde >", verde, "< azul >", azul, "< delta >", delta, "<")

	if rojo >= verde {
		valor = rojo
	} else {
		valor = verde
	}

	if azul >= valor {
		valor = azul
	}

	es_gris = false
	brillo = 255

	if rojo >= valor-delta {
		if verde >= valor-delta {
			if azul >= valor-delta {
				es_gris = true
				brillo = int(valor)
			}
		}
	}

	return es_gris, brillo
}

/*
Genera un diagrama de picos y valles horizontal
*/
func GenerarPicosVallesH(imagen_entrada image.RGBA, minimo_y int, maximo_y int) (imagen_pvh draw.Image) {

	var indice_x int
	var indice_y int
	var brillo_x []uint
	var ancho_x int
	var alto_y int
	var media_x int

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	ancho_x = max_x - min_x
	media_x = ancho_x / 2

	if (minimo_y == 0) && (maximo_y == 0) {
		minimo_y = min_y
		maximo_y = max_y
	}

	alto_y = maximo_y - minimo_y

	trace.Salida("GenerarPicosVallesH", 0, "  minimo_y <", minimo_y, "< maximo_y >", maximo_y, "< alto_y >", alto_y, "<")

	imagen_pvh = CrearImagenCualquierColor(ancho_x, maximo_y-minimo_y, color.Black)
	brillo_x = make([]uint, ancho_x)
	var sumar uint

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = minimo_y; indice_y <= maximo_y; indice_y++ {
			if ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y) > 128 { // La mitad del brillo
				sumar = 1
			} else {
				sumar = 0
			}
			brillo_x[indice_x] += sumar
			//			if indice_x == 146 {
			//				trace.Salida("GenerarPicosVallesH", 0, "  indice_x <", indice_x, "< indice_y >", indice_y, "< brillo_x >", brillo_x[indice_x], "<")
			//			}
		}
	}

	/*
		for indice_x = 0; indice_x < ancho_x; indice_x++ {
			brillo_x[indice_x] = brillo_x[indice_x] / uint(alto_y)
		}
	*/

	//trace.Salida("GenerarPicosVallesH", 0, "  brillo_max_1 <", brillo_max_1, "< brillo_max_2 >", brillo_max_2, "<")

	for indice_x = 0; indice_x < ancho_x; indice_x++ {
		if brillo_x[indice_x] > 0 {
			LineaRgbaV(imagen_pvh, indice_x, 0, int(brillo_x[indice_x]), color.RGBA{255, 255, 255, 255})
		}
	}

	// Encuentra los extremos

	var ultimo_negro_inicial int
	var ultimo_negro_final int
	var ultimo_blanco_inicial int
	var ultimo_blanco_final int

	limite := uint((float64(alto_y) * 0.05)) // Calculo basado en las placas 2016, los caracteres realmente son el 60% aproximadamente centrados

	for indice_x = 0; indice_x < media_x; indice_x++ {
		if brillo_x[indice_x] <= limite {
			ultimo_negro_inicial = indice_x
		} else {
			if brillo_x[indice_x] >= uint(alto_y)-limite {
				ultimo_blanco_inicial = indice_x
			}
		}
	}

	for indice_x = ancho_x - 1; indice_x > media_x; indice_x-- {
		if brillo_x[indice_x] <= limite {
			ultimo_negro_final = indice_x
		} else {
			if brillo_x[indice_x] >= uint(alto_y)-limite {
				ultimo_blanco_final = indice_x
			}
		}
	}

	//trace.Salida("GenerarPicosVallesH", 0, "  indice_x <", indice_x, "< brillo_x[indice_x] >", brillo_x[indice_x], "<")

	trace.Salida("GenerarPicosVallesH", 0, "  ultimo_negro_inicial <", ultimo_negro_inicial, "< ultimo_blanco_inicial >", ultimo_blanco_inicial, "<")
	trace.Salida("GenerarPicosVallesH", 0, "  ultimo_negro_final <", ultimo_negro_final, "< ultimo_blanco_final >", ultimo_blanco_final, "<")

	return imagen_pvh
}

/*
Genera un diagrama de picos y valles vertical
*/
func GenerarPicosVallesV(imagen_entrada image.RGBA) (imagen_pvv draw.Image, brillo_max_1 int, brillo_max_2 int) {

	var indice_x int
	var indice_y int
	var brillo_y []uint

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	alto_y := max_y - min_y
	media_y := alto_y / 2

	imagen_pvv = CrearImagenCualquierColor(max_x-min_x, alto_y, color.Black)
	brillo_y = make([]uint, alto_y)

	var sumar uint
	for indice_y = min_y; indice_y < max_y; indice_y++ {
		for indice_x = min_x; indice_x < max_x; indice_x++ {
			if ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y) > 128 { // La mitad del brillo
				sumar = 1
			} else {
				sumar = 0
			}
			brillo_y[indice_y] += sumar
		}
	}

	/*
		for indice_y = 0; indice_y < alto_y; indice_y++ {
			brillo_y[indice_y] = brillo_y[indice_y] / uint(max_x)
		}
	*/

	limite := int((float64(alto_y) * 0.1)) // Calculo basado en las placas 2016, los caracteres realmente son el 60% aproximadamente centrados

	trace.Salida("GenerarPicosVallesV", 0, "  media_y <", media_y, "< limite  >", limite, "<")

	// Busco de la mitad hacia arriba
	brillo_max_1 = media_y
	for indice_y = media_y; indice_y >= limite; indice_y-- {
		if brillo_y[indice_y] >= brillo_y[brillo_max_1] {
			brillo_max_1 = indice_y
		}
		/*
			if indice_y == 28 {
				trace.Salida("GenerarPicosVallesV", 0, "  brillo_y[indice_y] <", brillo_y[indice_y], "< brillo_max_1 >", brillo_max_1, "<")
			}
		*/
	}

	// Busco de la mitad hacia abajo
	brillo_max_2 = media_y
	for indice_y = media_y - 1; indice_y < (alto_y - limite); indice_y++ {
		if brillo_y[indice_y] >= brillo_y[brillo_max_2] {
			brillo_max_2 = indice_y
		}
	}

	trace.Salida("GenerarPicosVallesV", 0, "  brillo_max_1 <", brillo_max_1, "< brillo_max_2 >", brillo_max_2, "<")

	for indice_y = 0; indice_y < max_y-min_y; indice_y++ {
		if brillo_y[indice_y] > 0 {
			LineaRgbaH(imagen_pvv, 0, indice_y, int(brillo_y[indice_y]), color.RGBA{255, 255, 255, 255})
		}
	}

	return imagen_pvv, brillo_max_1, brillo_max_2
}

/*
Filtro numero 7
Filtro para obtener los bordes de una imagen a colores y crea una imagen con solo dos colores
*/
func SobelNegrosBin(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	var valor int
	var valor2 int
	var color_calculado uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenGris(max_x-min_x, max_y-min_y)

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {

			/*

			   Matriz a trabajar

			    1  2  1
			    0  X  0
			   -1 -2 -1

			*/

			valor = int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y-1))
			valor += 2 * int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y-1))
			valor += int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y-1))
			valor -= int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y+1))
			valor -= 2 * int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y+1))
			valor -= int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y+1))

			if valor < 0 {
				valor = -1 * valor
			}

			/*
			   Matriz a trabajar

			   -1 0 1
			   -2 X 2
			   -1 0 1

			*/

			valor2 = int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y-1))
			valor2 += 2 * int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y))
			valor2 += int(ImagenBrillantezRGB(imagen_entrada, indice_x+1, indice_y+1))
			valor2 -= int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y-1))
			valor2 -= 2 * int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y))
			valor2 -= int(ImagenBrillantezRGB(imagen_entrada, indice_x-1, indice_y+1))

			if valor2 < 0 {
				valor2 = -1 * valor2
			}

			valor += valor2
			valor = valor / 2

			//color_calculado = uint8(valor)
			if valor > 128 {
				color_calculado = 255
			} else {
				color_calculado = 0
			}
			imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})

			//trace.Salida("SobelNegros", 0, "  indice_x <", indice_x, "< indice_y >", indice_y, "< valor >", valor, "<")

		}
	}

	return imagen_sin_colores
}

/*
Obtiene la brillantez de un punto en una imágen
regresa brillo entre 0 y 255
*/
func ImagenBrillantezRGB(imagen image.RGBA, x int, y int) (brillantez uint) {

	rojo, verde, azul, _ := imagen.At(x, y).RGBA()

	rojo = rojo >> 8
	verde = verde >> 8
	azul = azul >> 8

	if (rojo == verde) && (verde == azul) {
		brillantez = uint(rojo)
	} else {

		valor_brillantez_rojo := rojo * relacion_brillantez_rojo
		valor_brillantez_verde := verde * relacion_brillantez_verde
		valor_brillantez_azul := azul * relacion_brillantez_azul

		/* El valor 125 es equivalente a sumarle un 49%, para redondear al siguiente numero */

		brillantez = uint((valor_brillantez_rojo + valor_brillantez_verde + valor_brillantez_azul) >> 8)
	}

	return brillantez
}

/*
Obtiene la brillantez de un punto en una imágen gris
regresa brillo entre 0 y 255
*/
func ImagenBrillantezGris(imagen draw.Image, x int, y int) (brillantez uint) {

	rojo, _, _, _ := imagen.At(x, y).RGBA()

	rojo = rojo >> 8

	brillantez = uint(rojo)

	return brillantez
}

/*
Rota una imagen por el angulo en grados
El angulo es en grados multiplicado por 10, ej 34.2 llega como 342
direccion en 1 indica que la rotacion es contra reloj y -1 es reloj
*/
func RotarImagen(imagen_entrada image.RGBA, angulo int, direccion int) (imagen_rotada draw.Image) {

	var ang_rotacion int
	var largo_imagen int
	//var ancho_imagen int
	var delta_x []int
	var delta_y []int
	var inicio_x int
	var inicio_y int
	var indice int
	var indice_x int
	var indice_y int
	var color_original color.Color
	var tam_max_x int
	var tam_max_y int
	var indice_probar int
	var dst draw.Image

	trace.Salida("RotarImagen", 0, "angulo >", angulo, "< direccion >", direccion, "<")

	if angulo > 1800 {
		ang_rotacion = angulo - 1800
	} else {
		ang_rotacion = angulo // OJO CGC
	}

	//ang_rotacion *= 10

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	largo_imagen = max_x - min_x

	trace.Salida("RotarImagen", 0, "largo_imagen >", largo_imagen, "<")

	delta_x = make([]int, largo_imagen)
	delta_y = make([]int, largo_imagen)

	for indice = min_x; indice < max_x; indice++ {
		if direccion == 1 { // Direccion contrareloj
			r := float64(indice) / cos_angulo[ang_rotacion]
			delta_x[indice-min_x] = int(math.Round(r)) - indice
			delta_y[indice-min_x] = -1 * int(math.Round(r*sen_angulo[ang_rotacion]))
		} else {
			r := float64(indice) / cos_angulo[ang_rotacion]
			delta_x[indice-min_x] = int(math.Round(r)) - indice
			delta_y[indice-min_x] = int(math.Round(r * sen_angulo[ang_rotacion]))
		}

		trace.Salida("RotarImagen", 0, "indice >", indice, "< delta_x >", delta_x[indice-min_x], "< x no int >", float64(indice)*(1-cos_angulo[ang_rotacion]), "<")
		trace.Salida("RotarImagen", 0, "indice >", indice, "< delta_y >", delta_y[indice-min_x], "< y no int >", float64(indice)*sen_angulo[ang_rotacion], "<")
	}

	if direccion == 1 {
		indice_probar = max_x - 1
	} else {
		indice_probar = 0
	}
	if delta_x[indice_probar] < 0 {
		tam_max_x = -delta_x[indice_probar]
	} else {
		tam_max_x = delta_x[indice_probar]
	}
	if delta_y[max_x-1] < 0 {
		tam_max_y = -delta_y[max_x-1]
	} else {
		tam_max_y = delta_y[max_x-1]
	}

	//	trace.Salida("RotarImagen", 0, "tam_max_x >", tam_max_x, "< tam_max_y >", tam_max_y, "<")

	imagen_rotada = CrearImagenColor(max_x+tam_max_x, max_y+tam_max_y)

	if direccion == 1 {
		inicio_x = 0
		inicio_y = tam_max_y
	} else {
		inicio_x = tam_max_x
		inicio_y = 0
	}

	var anterior int

	for indice_y = min_y; indice_y < max_y; indice_y++ {
		anterior = 0
		for indice_x = min_x; indice_x < max_x; indice_x++ {
			color_original = imagen_entrada.At(indice_x, indice_y)

			//			trace.Salida("RotarImagen", 0, "> anterior <", anterior, inicio_x+(delta_x[indice_x-min_x]), "<")

			if anterior != indice_x+(delta_x[indice_x-min_x]) {
				anterior++
				imagen_rotada.Set(inicio_x+indice_x+(delta_x[indice_x-min_x])-1, inicio_y+indice_y+(delta_y[indice_x-min_x]), color_original)
			}

			anterior++
			/*
				if indice_y == 50 {
					trace.Salida("RotarImagen", 0, "indice_x >", indice_x, "< indice_y >", indice_y, "< d x >", delta_x[indice_x-min_x], "< d y >", delta_y[indice_x-min_x], "<")
					trace.Salida("RotarImagen", 0, "indice_x >", indice_x, "< indice_y >", indice_y, "< x >", inicio_x+indice_x+(delta_x[indice_x-min_x]), "< y >", inicio_y+indice_y+(delta_y[indice_x-min_x]), "<")
				}
			*/
			imagen_rotada.Set(inicio_x+indice_x+(delta_x[indice_x-min_x]), inicio_y+indice_y+(delta_y[indice_x-min_x]), color_original) // OJO CGC pensar si no me conviene extender la imagen
		}
	}

	//largo_imagen = delta_x[max_x-1]
	//ancho_imagen = max_y + delta_y[max_x-1] - 2*inicio_y

	//	LineaRgbaH(imagen_rotada, 0, inicio_y, max_y+delta_y[max_x-1], color.RGBA{0, 255, 0, 255})
	//	LineaRgbaH(imagen_rotada, 0, max_y+delta_y[max_x-1]-inicio_y, max_y+delta_y[max_x-1], color.RGBA{0, 255, 0, 255})
	// RectanguloRgba(imagen_rotada, 0, inicio_y, delta_x[max_x-1]-1, max_y+delta_y[max_x-1]-inicio_y, color.RGBA{0, 255, 0, 255})

	// Recortar la imagen original

	dst = image.NewRGBA(image.Rect(0, 0, max_x+tam_max_x, max_y-tam_max_y))
	if direccion == 1 {
		draw.Draw(dst, dst.Bounds(), imagen_rotada, image.Point{0, tam_max_y}, draw.Src)
	} else {
		draw.Draw(dst, dst.Bounds(), imagen_rotada, image.Point{0, tam_max_y}, draw.Src)
	}

	return dst
}

// LineaH dibuja una linea horizontal
func LineaRgbaH(imagen draw.Image, x1 int, y int, x2 int, col color.RGBA) {
	for ; x1 <= x2; x1++ {
		imagen.Set(x1, y, col)
	}
}

// LineaV dibuja una linea vertical
func LineaRgbaV(imagen draw.Image, x int, y1 int, y2 int, col color.RGBA) {
	for ; y1 <= y2; y1++ {
		imagen.Set(x, y1, col)
	}
}

// Rectangulo dibuja un rectangulo
func RectanguloRgba(imagen draw.Image, x1 int, y1 int, x2 int, y2 int, col color.RGBA) {
	LineaRgbaH(imagen, x1, y1, x2, col)
	LineaRgbaH(imagen, x1, y2, x2, col)
	LineaRgbaV(imagen, x1, y1, y2, col)
	LineaRgbaV(imagen, x2, y1, y2, col)
}

func CrearImagenGris(max_x, max_y int) draw.Image {

	him := image.NewGray16(image.Rect(0, 0, max_x, max_y))
	draw.Draw(him, him.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	return him
}

/*
Crea una imagen nueva con fondo blanco
*/
func CrearImagenColor(max_x, max_y int) draw.Image {

	him := image.NewRGBA(image.Rect(0, 0, max_x, max_y))
	draw.Draw(him, him.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	return him
}

/*
Crea una imagen nueva con fondo de cualquier color
*/
func CrearImagenCualquierColor(max_x, max_y int, color_nuevo color.Color) draw.Image {

	him := image.NewRGBA(image.Rect(0, 0, max_x, max_y))
	draw.Draw(him, him.Bounds(), image.NewUniform(color_nuevo),
		image.Point{}, draw.Src)

	return him
}

/*

	Filtro dos colores
	 https://en.wikipedia.org/wiki/Otsu%27s_method

*/

/*
Fitro numero 6
Filtro para crear un borde
*/
func NuevoFiltroSobel(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var valor int
	var valor2 int
	var grid_externa_x int
	var grid_externa_y int
	var grid_interna_x int
	var grid_interna_y int
	var maximo_grid int
	var minimo_grid int
	var valor_brillo int
	var delta_brillo int
	var color_calculado uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)

	for grid_externa_y = min_y + 1; grid_externa_y < max_y-5; grid_externa_y += 3 {
		for grid_externa_x = min_x + 1; grid_externa_x < max_x-5; grid_externa_x += 3 {
			maximo_grid = 0
			minimo_grid = 255
			valor_brillo = 0
			for grid_interna_y = grid_externa_y; grid_interna_y < (grid_externa_y + 5); grid_interna_y++ {
				for grid_interna_x = grid_externa_x; grid_interna_x < (grid_externa_x + 5); grid_interna_x++ {
					valor_brillo = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y))

					if valor_brillo >= maximo_grid {
						maximo_grid = valor_brillo
					}
					if valor_brillo <= minimo_grid {
						minimo_grid = valor_brillo
					}
				}
			}

			delta_brillo = maximo_grid - minimo_grid

			//			trace.Salida("NuevoFiltroSobel", 0, "maximo_grid >", maximo_grid, "< minimo_grid >", minimo_grid, "< delta_brillo >", delta_brillo, "<")

			if delta_brillo >= 26 { // El diferencial debera ser superior al 10% del brillo para hacer el aa1lisis

				/* Proceso de los pixeles interiores a la grid 5 X 5 */

				valor = 0
				valor2 = 0

				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						/*
							if FiltroEliminarPixel(imagen_entrada, grid_interna_x, grid_interna_y) == "NO" {
						*/

						/*

						  Matriz a trabajar

						   1  2  1
						   0  X  0
						  -1 -2 -1

						*/

						valor = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y-1))
						valor += 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y-1))
						valor += int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y-1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y+1))
						valor -= 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y+1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y+1))

						if valor < 0 {
							valor = -1 * valor
						}

						/*
						   Matriz a trabajar

						   -1 0 1
						   -2 X 2
						   -1 0 1

						*/

						valor2 = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y-1))
						valor2 += 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y))
						valor2 += int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y+1))
						valor2 -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y-1))
						valor2 -= 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y))
						valor2 -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y+1))

						if valor2 < 0 {
							valor2 = -1 * valor2
						}

						valor += valor2

						// if valor >= 204 // CGC Sobel oscuros traidcional
						if valor >= (delta_brillo * 2) {
							valor = 255
						} else {
							valor = 0
						}

						color_calculado = uint8(valor) // CGC debo de usar los bits altos

						/* CGC Si quiero ver unicamente lo que estoy quitando descomento esta linea */
						//color_calculado = 0;

						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})

						/*
							} else {
								// CGC Si quiero ver unicamente lo que estoy quitando comento esta linea
								imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})

							}
						*/
					}
				}
			} else {
				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}

	/* Ponemos en negro los pixeles de los bordes */

	RectanguloRgba(imagen_sin_colores, min_x, min_y, max_x, max_y, color.RGBA{0, 0, 0, 255})

	trace.Salida("NuevoFiltroSobel", 0, "Termino")

	return imagen_sin_colores
}

/*
Filtro para preparar la imagen original la binariza y le agrega borde
*/
func PreparaPaso2(imagen_entrada image.RGBA, borde int) (imagen_sin_colores draw.Image) {

	var indice_x int
	var indice_y int
	var valor int
	var color_calculado uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenColor(max_x-min_x+borde, max_y-min_y+borde)

	for indice_x = min_x + borde; indice_x < max_x; indice_x++ {
		for indice_y = min_y + borde; indice_y < max_y; indice_y++ {
			valor = int(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y))
			if valor >= 128 {
				valor = 255
			} else {
				valor = 0
			}

			color_calculado = uint8(valor)
			imagen_sin_colores.Set(indice_x, indice_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})
		}
	}

	return imagen_sin_colores
}

//////////////////////////////////////////////////////////////////////////////
//
//  Funcion:      FiltroEliminarPixel
//  Descripcion:  Funcion que toma un pixel y le aplica criterios para decidir
//                si se debe no analizarlo, regresa SI o NO
//
//////////////////////////////////////////////////////////////////////////////
/*
func FiltroEliminarPixel(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {
}
*/

/*
Genera un diagrama de picos y valles vertical
Aqui busco desde el inicio de y por lo que si hay picos falsos los toma
*/
func GenerarPicosVallesV_v1(imagen_entrada image.RGBA) (imagen_pvv draw.Image, brillo_max_1 int, brillo_max_2 int) {

	var indice_x int
	var indice_y int
	var brillo_y []uint

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_pvv = CrearImagenColor(256, max_y-min_y)

	brillo_y = make([]uint, max_y-min_y)
	for indice_y = min_y; indice_y < max_y; indice_y++ {
		for indice_x = min_x; indice_x < max_x; indice_x++ {
			brillo_y[indice_y-min_y] += ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y)
		}
	}

	brillo_max_1 = 0
	for indice_y = 0; indice_y < max_y-min_y; indice_y++ {
		brillo_y[indice_y] = brillo_y[indice_y] / uint(max_x)
		if brillo_y[indice_y] > brillo_y[brillo_max_1] {
			brillo_max_1 = indice_y
		}
	}

	brillo_max_2 = 0
	for indice_y = 0; indice_y < max_y-min_y; indice_y++ {
		trace.Salida("GenerarPicosVallesV", 0, "  brillo_max_1 <", brillo_max_1, "< indice_y >", indice_y, "< brillo_y[indice_y] >", brillo_y[indice_y], "<")
		if (brillo_y[indice_y] > brillo_y[brillo_max_2]) && (indice_y != brillo_max_1) {
			brillo_max_2 = indice_y
		}
	}

	trace.Salida("GenerarPicosVallesV", 0, "  brillo_max_1 <", brillo_max_1, "< brillo_max_2 >", brillo_max_2, "<")

	for indice_y = 0; indice_y < max_y-min_y; indice_y++ {
		LineaRgbaH(imagen_pvv, 0, indice_y, int(brillo_y[indice_y]), color.RGBA{255, 0, 255, 255})
	}

	return imagen_pvv, brillo_max_1, brillo_max_2
}

/*
Agrega un marco a los textos
*/
func Marco(imagen_entrada image.RGBA, borde int) (imagen_salida draw.Image) {

	var indice_x int
	var indice_y int

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_salida = CrearImagenColor(max_x-min_x+(borde*2), max_y-min_y+(borde*2))

	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {
			col := imagen_entrada.At(indice_x, indice_y)
			imagen_salida.Set(indice_x+borde, indice_y+borde, col)
		}
	}

	return imagen_salida
}

/*
Fitro numero 17
Filtro para deteccion de borde horizontal
*/
func SobelH1(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var valor int
	var grid_externa_x int
	var grid_externa_y int
	var grid_interna_x int
	var grid_interna_y int
	var maximo_grid int
	var minimo_grid int
	var valor_brillo int
	var delta_brillo int
	var color_calculado uint8
	// var brillo_central uint8
	var indice_x int
	var indice_y int
	var distribucion_brillo []uint8
	var brillantez []int64
	var cuantos_puntos int64
	var brillo uint8

	distribucion_brillo = make([]uint8, 10)
	brillantez = make([]int64, 256)

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	cuantos_puntos = int64((max_y - min_y) * (max_x - min_x))
	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)

	// Crea imagen en grises
	for indice_x = min_x; indice_x < max_x; indice_x++ {
		for indice_y = min_y; indice_y < max_y; indice_y++ {
			brillo = uint8(ImagenBrillantezRGB(imagen_entrada, indice_x, indice_y))
			brillantez[int(brillo)]++
		}
	}

	// Encuentra el 10%
	var contador int64
	var decimos int64 = 1
	var indice uint8
	for indice = 0; indice < 255; indice++ {
		contador += brillantez[indice]
		if contador >= (cuantos_puntos*decimos)/10 {
			decimos = int64(contador / (cuantos_puntos / 10))
			distribucion_brillo[decimos-1] = indice - 1
			decimos++
		}
	}
	/*
		for indice = 0; indice < 10; indice++ {
			trace.Salida("SobelH", 0, "indice >", indice, "< distribucion_brillo >", distribucion_brillo[indice], "<")
		}
	*/
	for grid_externa_y = min_y + 1; grid_externa_y < max_y-5; grid_externa_y += 3 {
		for grid_externa_x = min_x + 1; grid_externa_x < max_x-5; grid_externa_x += 3 {
			maximo_grid = 0
			minimo_grid = 255
			valor_brillo = 0
			for grid_interna_y = grid_externa_y; grid_interna_y < (grid_externa_y + 5); grid_interna_y++ {
				for grid_interna_x = grid_externa_x; grid_interna_x < (grid_externa_x + 5); grid_interna_x++ {
					valor_brillo = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y))

					if valor_brillo >= maximo_grid {
						maximo_grid = valor_brillo
					}
					if valor_brillo <= minimo_grid {
						minimo_grid = valor_brillo
					}
				}
			}

			delta_brillo = maximo_grid - minimo_grid

			//			trace.Salida("NuevoFiltroSobel", 0, "maximo_grid >", maximo_grid, "< minimo_grid >", minimo_grid, "< delta_brillo >", delta_brillo, "<")

			if delta_brillo >= 26 { // El diferencial debera ser superior al 10% del brillo para hacer el aa1lisis

				/* Proceso de los pixeles interiores a la grid 5 X 5 */

				valor = 0

				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						/*
							if FiltroEliminarPixel(imagen_entrada, grid_interna_x, grid_interna_y) == "NO" {
						*/

						//brillo_central = uint8(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y))

						/*

						  Matriz a trabajar

						   1  2  1
						   0  X  0
						  -1 -2 -1

						*/

						valor = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y-1))
						valor += 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y-1))
						valor += int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y-1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y+1))
						valor -= 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y+1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y+1))

						if valor < 0 {
							valor = -1 * valor
						}

						/*

							if valor > int(brillo_central) {
								color_calculado = 255
							} else {
								color_calculado = 0
							}

							if grid_interna_y == 23 {
								trace.Salida("SobelH", 0, "valor >", valor, "< brillo_central >", brillo_central, "< color_calculado >", color_calculado, "<")
							}

								valor /= int(brillo_central)

								if brillo_central < distribucion_brillo[8] { // Es negro
									color_calculado = 0
								} else {
									if brillo_central > distribucion_brillo[7] { // Es blanco
										color_calculado = 255
									} else {
										if valor < 3 { // Es una planicie blanca
											color_calculado = 255
										} else { // Es un borde
											color_calculado = 0
										}
									}
								}
						*/

						if valor >= (delta_brillo * 2) {
							valor = 255
						} else {
							valor = 0
						}

						color_calculado = uint8(valor) // CGC debo de usar los bits altos

						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})
					}
				}
				/*
					} else {
						for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
							for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
								imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})
							}
						}
				*/
			}
		}
	}

	/* Ponemos en negro los pixeles de los bordes */

	//RectanguloRgba(imagen_sin_colores, min_x, min_y, max_x, max_y, color.RGBA{0, 0, 0, 255})

	trace.Salida("SobelH", 0, "Termino")

	return imagen_sin_colores
}

/*
Fitro numero 17
Filtro para deteccion de borde vertical
*/
func SobelH(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var valor int
	var grid_externa_x int
	var grid_externa_y int
	var grid_interna_x int
	var grid_interna_y int
	var maximo_grid int
	var minimo_grid int
	var valor_brillo int
	var delta_brillo int
	var color_calculado uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)

	for grid_externa_y = min_y + 1; grid_externa_y < max_y-5; grid_externa_y += 3 {
		for grid_externa_x = min_x + 1; grid_externa_x < max_x-5; grid_externa_x += 3 {
			maximo_grid = 0
			minimo_grid = 255
			valor_brillo = 0
			for grid_interna_y = grid_externa_y; grid_interna_y < (grid_externa_y + 5); grid_interna_y++ {
				for grid_interna_x = grid_externa_x; grid_interna_x < (grid_externa_x + 5); grid_interna_x++ {
					valor_brillo = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y))

					if valor_brillo >= maximo_grid {
						maximo_grid = valor_brillo
					}
					if valor_brillo <= minimo_grid {
						minimo_grid = valor_brillo
					}
				}
			}

			delta_brillo = maximo_grid - minimo_grid

			//			trace.Salida("NuevoFiltroSobel", 0, "maximo_grid >", maximo_grid, "< minimo_grid >", minimo_grid, "< delta_brillo >", delta_brillo, "<")

			if delta_brillo >= 26 { // El diferencial debera ser superior al 10% del brillo para hacer el aa1lisis

				/* Proceso de los pixeles interiores a la grid 5 X 5 */

				valor = 0

				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						/*
							if FiltroEliminarPixel(imagen_entrada, grid_interna_x, grid_interna_y) == "NO" {
						*/

						/*

						  Matriz a trabajar

						   1  2  1
						   0  X  0
						  -1 -2 -1

						*/

						valor = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y-1))
						valor += 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y-1))
						valor += int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y-1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y+1))
						valor -= 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y+1))
						valor -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y+1))

						if valor < 0 {
							valor = -1 * valor
						}

						// if valor >= 204 // CGC Sobel oscuros traidcional
						if valor >= (delta_brillo * 2) {
							valor = 255
						} else {
							valor = 0
						}

						color_calculado = uint8(valor) // CGC debo de usar los bits altos

						/* CGC Si quiero ver unicamente lo que estoy quitando descomento esta linea */
						//color_calculado = 0;

						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})

						/*
							} else {
								// CGC Si quiero ver unicamente lo que estoy quitando comento esta linea
								imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})

							}
						*/
					}
				}
			} else {
				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}

	/* Ponemos en negro los pixeles de los bordes */

	RectanguloRgba(imagen_sin_colores, min_x, min_y, max_x, max_y, color.RGBA{0, 0, 0, 255})

	trace.Salida("NuevoFiltroSobel", 0, "Termino")

	return imagen_sin_colores
}

/*
Fitro numero 18
Filtro para deteccion de borde vertical
*/
func SobelV(imagen_entrada image.RGBA) (imagen_sin_colores draw.Image) {

	var valor int
	var valor2 int
	var grid_externa_x int
	var grid_externa_y int
	var grid_interna_x int
	var grid_interna_y int
	var maximo_grid int
	var minimo_grid int
	var valor_brillo int
	var delta_brillo int
	var color_calculado uint8

	min_x := imagen_entrada.Bounds().Min.X
	min_y := imagen_entrada.Bounds().Min.Y
	max_x := imagen_entrada.Bounds().Max.X
	max_y := imagen_entrada.Bounds().Max.Y

	imagen_sin_colores = CrearImagenColor(max_x-min_x, max_y-min_y)

	for grid_externa_y = min_y + 1; grid_externa_y < max_y-5; grid_externa_y += 3 {
		for grid_externa_x = min_x + 1; grid_externa_x < max_x-5; grid_externa_x += 3 {
			maximo_grid = 0
			minimo_grid = 255
			valor_brillo = 0
			for grid_interna_y = grid_externa_y; grid_interna_y < (grid_externa_y + 5); grid_interna_y++ {
				for grid_interna_x = grid_externa_x; grid_interna_x < (grid_externa_x + 5); grid_interna_x++ {
					valor_brillo = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x, grid_interna_y))

					if valor_brillo >= maximo_grid {
						maximo_grid = valor_brillo
					}
					if valor_brillo <= minimo_grid {
						minimo_grid = valor_brillo
					}
				}
			}

			delta_brillo = maximo_grid - minimo_grid

			//			trace.Salida("NuevoFiltroSobel", 0, "maximo_grid >", maximo_grid, "< minimo_grid >", minimo_grid, "< delta_brillo >", delta_brillo, "<")

			if delta_brillo >= 26 { // El diferencial debera ser superior al 10% del brillo para hacer el aa1lisis

				/* Proceso de los pixeles interiores a la grid 5 X 5 */

				valor = 0
				valor2 = 0

				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						/*
							if FiltroEliminarPixel(imagen_entrada, grid_interna_x, grid_interna_y) == "NO" {
						*/

						/*
						   Matriz a trabajar

						   -1 0 1
						   -2 X 2
						   -1 0 1

						*/

						valor2 = int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y-1))
						valor2 += 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y))
						valor2 += int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x+1, grid_interna_y+1))
						valor2 -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y-1))
						valor2 -= 2 * int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y))
						valor2 -= int(ImagenBrillantezRGB(imagen_entrada, grid_interna_x-1, grid_interna_y+1))

						if valor2 < 0 {
							valor2 = -1 * valor2
						}

						valor = valor2

						// if valor >= 204 // CGC Sobel oscuros traidcional
						if valor >= (delta_brillo * 2) {
							valor = 255
						} else {
							valor = 0
						}

						color_calculado = uint8(valor) // CGC debo de usar los bits altos

						/* CGC Si quiero ver unicamente lo que estoy quitando descomento esta linea */
						//color_calculado = 0;

						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{color_calculado, color_calculado, color_calculado, 255})

						/*
							} else {
								// CGC Si quiero ver unicamente lo que estoy quitando comento esta linea
								imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})

							}
						*/
					}
				}
			} else {
				for grid_interna_y = grid_externa_y + 1; grid_interna_y < (grid_externa_y + 4); grid_interna_y++ {
					for grid_interna_x = grid_externa_x + 1; grid_interna_x < (grid_externa_x + 4); grid_interna_x++ {
						imagen_sin_colores.Set(grid_interna_x, grid_interna_y, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}

	/* Ponemos en negro los pixeles de los bordes */

	RectanguloRgba(imagen_sin_colores, min_x, min_y, max_x, max_y, color.RGBA{0, 0, 0, 255})

	trace.Salida("NuevoFiltroSobel", 0, "Termino")

	return imagen_sin_colores
}
