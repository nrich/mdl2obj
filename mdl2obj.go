/*
 * mdl2obj
 *
 * Copyright (C) 2016-2019 Florian Zwoch <fzwoch@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type vec3 struct {
	X float32
	Y float32
	Z float32
}

type mdlHeader struct {
	ID           [4]byte
	Version      uint32
	Scale        vec3
	Origin       vec3
	Radius       float32
	Offsets      vec3
	NumSkins     uint32
	SkinWidth    uint32
	SkinHeight   uint32
	NumVerts     uint32
	NumTriangles uint32
	NumFrames    uint32
	SyncType     uint32
	Flags        uint32
	Size         float32
}

type skin struct {
	Type uint32
}

type skinGroup struct {
	NumSkins uint32
	Time     float32
}

type stVert struct {
	OnSeam uint32
	S      uint32
	T      uint32
}

type triangle struct {
	Front  uint32
	Vertex [3]uint32
}

type vert struct {
	V      [3]uint8
	Normal uint8
}

type frame struct {
	Type uint32
	Min  vert
	Max  vert
	Name [16]uint8
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage:", os.Args[0], "<input.mdl>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	mdlName := filepath.Base(strings.TrimSuffix(os.Args[1], filepath.Ext(os.Args[1])))

	var mdl mdlHeader

	err = binary.Read(f, binary.LittleEndian, &mdl)
	if err != nil {
		panic(err)
	}

	if string(mdl.ID[:]) != "IDPO" {
		panic("MDL magic " + string(mdl.ID[:]) + " != IDPO")
	}

	if mdl.Version != 6 {
		panic("MDL version " + string(mdl.Version) + " != 6")
	}

	for i := 0; i < int(mdl.NumSkins); i++ {
		var skin skin

		binary.Read(f, binary.LittleEndian, &skin)

		skinGroup := skinGroup{
			NumSkins: 1,
		}

		if skin.Type != 0 {
			binary.Read(f, binary.LittleEndian, &skinGroup)
		}

		f.Seek(int64(skinGroup.NumSkins*mdl.SkinWidth*mdl.SkinHeight), io.SeekCurrent)
	}

	stverts := make([]stVert, mdl.NumVerts)
	triangles := make([]triangle, mdl.NumTriangles)

	binary.Read(f, binary.LittleEndian, &stverts)
	binary.Read(f, binary.LittleEndian, &triangles)

	verts := [][]vert{
		make([]vert, mdl.NumFrames),
	}

	for i := range verts {
		var frame frame

		binary.Read(f, binary.LittleEndian, &frame)

		if frame.Type != 0 {
			panic("FIXME")
		}

                fmt.Println(frame.Name)

		verts[i] = make([]vert, mdl.NumVerts)
		binary.Read(f, binary.LittleEndian, &verts[i])
	}

	data := struct {
		Name string
		V    [][3]float32
		VT   [][2]float32
		F    [][9]uint32
	}{
		Name: mdlName,
		V:    make([][3]float32, len(verts[0])),
		VT:   make([][2]float32, len(stverts)*2),
		F:    make([][9]uint32, len(triangles)),
	}

	for i, v := range verts[0] {
		data.V[i] = [3]float32{
			-mdl.Scale.X*float32(v.V[0]) - mdl.Origin.X,
			mdl.Scale.Z*float32(v.V[2]) + mdl.Origin.Z,
			mdl.Scale.Y*float32(v.V[1]) + mdl.Origin.Y,
		}
	}

	for i, st := range stverts {
		data.VT[i] = [2]float32{
			float32(st.S) / float32(mdl.SkinWidth),
			float32(1 - float32(st.T)/float32(mdl.SkinHeight)),
		}
		data.VT[i+int(mdl.NumVerts)] = [2]float32{
			(float32(st.S) + float32(mdl.SkinWidth/2)) / float32(mdl.SkinWidth),
			1 - float32(st.T)/float32(mdl.SkinHeight),
		}
	}

	for i, t := range triangles {
		var uv [3]uint32

		for i, v := range t.Vertex {
			uv[i] = v
			if t.Front == 0 && stverts[v].OnSeam != 0 {
				uv[i] += mdl.NumVerts
			}
		}

		data.F[i] = [9]uint32{
			t.Vertex[0] + 1,
			uv[0] + 1,
                        uint32(verts[0][t.Vertex[0]].Normal),
			t.Vertex[2] + 1,
			uv[2] + 1,
                        uint32(verts[0][t.Vertex[1]].Normal),
			t.Vertex[1] + 1,
			uv[1] + 1,
                        uint32(verts[0][t.Vertex[2]].Normal),
		}
	}

	o, err := os.Create(mdlName + ".obj")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(o)

	t, _ := template.New("").Parse(obj)
	t.Execute(w, data)
	w.Flush()
	o.Close()

	o, err = os.Create(mdlName + ".mtl")
	if err != nil {
		panic(err)
	}
	w = bufio.NewWriter(o)

	t, _ = template.New("").Parse(mtl)
	t.Execute(w, data)
	w.Flush()
	o.Close()
}

const obj = `o {{ .Name }}
mtllib {{ .Name }}.mtl
usemtl {{ .Name }}
{{range .V}}v {{ index . 0 }} {{ index . 1 }} {{ index . 2 }}
{{end}}vn -0.525731  0.000000  0.850651  
vn -0.442863  0.238856  0.864188  
vn -0.295242  0.000000  0.955423  
vn -0.309017  0.500000  0.809017  
vn -0.162460  0.262866  0.951056  
vn  0.000000  0.000000  1.000000  
vn  0.000000  0.850651  0.525731  
vn -0.147621  0.716567  0.681718  
vn  0.147621  0.716567  0.681718  
vn  0.000000  0.525731  0.850651  
vn  0.309017  0.500000  0.809017  
vn  0.525731  0.000000  0.850651  
vn  0.295242  0.000000  0.955423  
vn  0.442863  0.238856  0.864188  
vn  0.162460  0.262866  0.951056  
vn -0.681718  0.147621  0.716567  
vn -0.809017  0.309017  0.500000  
vn -0.587785  0.425325  0.688191  
vn -0.850651  0.525731  0.000000  
vn -0.864188  0.442863  0.238856  
vn -0.716567  0.681718  0.147621  
vn -0.688191  0.587785  0.425325  
vn -0.500000  0.809017  0.309017  
vn -0.238856  0.864188  0.442863  
vn -0.425325  0.688191  0.587785  
vn -0.716567  0.681718 -0.147621  
vn -0.500000  0.809017 -0.309017  
vn -0.525731  0.850651  0.000000  
vn  0.000000  0.850651 -0.525731  
vn -0.238856  0.864188 -0.442863  
vn  0.000000  0.955423 -0.295242  
vn -0.262866  0.951056 -0.162460  
vn  0.000000  1.000000  0.000000  
vn  0.000000  0.955423  0.295242  
vn -0.262866  0.951056  0.162460  
vn  0.238856  0.864188  0.442863  
vn  0.262866  0.951056  0.162460  
vn  0.500000  0.809017  0.309017  
vn  0.238856  0.864188 -0.442863  
vn  0.262866  0.951056 -0.162460  
vn  0.500000  0.809017 -0.309017  
vn  0.850651  0.525731  0.000000  
vn  0.716567  0.681718  0.147621  
vn  0.716567  0.681718 -0.147621  
vn  0.525731  0.850651  0.000000  
vn  0.425325  0.688191  0.587785  
vn  0.864188  0.442863  0.238856  
vn  0.688191  0.587785  0.425325  
vn  0.809017  0.309017  0.500000  
vn  0.681718  0.147621  0.716567  
vn  0.587785  0.425325  0.688191  
vn  0.955423  0.295242  0.000000  
vn  1.000000  0.000000  0.000000  
vn  0.951056  0.162460  0.262866  
vn  0.850651 -0.525731  0.000000  
vn  0.955423 -0.295242  0.000000  
vn  0.864188 -0.442863  0.238856  
vn  0.951056 -0.162460  0.262866  
vn  0.809017 -0.309017  0.500000  
vn  0.681718 -0.147621  0.716567  
vn  0.850651  0.000000  0.525731  
vn  0.864188  0.442863 -0.238856  
vn  0.809017  0.309017 -0.500000  
vn  0.951056  0.162460 -0.262866  
vn  0.525731  0.000000 -0.850651  
vn  0.681718  0.147621 -0.716567  
vn  0.681718 -0.147621 -0.716567  
vn  0.850651  0.000000 -0.525731  
vn  0.809017 -0.309017 -0.500000  
vn  0.864188 -0.442863 -0.238856  
vn  0.951056 -0.162460 -0.262866  
vn  0.147621  0.716567 -0.681718  
vn  0.309017  0.500000 -0.809017  
vn  0.425325  0.688191 -0.587785  
vn  0.442863  0.238856 -0.864188  
vn  0.587785  0.425325 -0.688191  
vn  0.688191  0.587785 -0.425325  
vn -0.147621  0.716567 -0.681718  
vn -0.309017  0.500000 -0.809017  
vn  0.000000  0.525731 -0.850651  
vn -0.525731  0.000000 -0.850651  
vn -0.442863  0.238856 -0.864188  
vn -0.295242  0.000000 -0.955423  
vn -0.162460  0.262866 -0.951056  
vn  0.000000  0.000000 -1.000000  
vn  0.295242  0.000000 -0.955423  
vn  0.162460  0.262866 -0.951056  
vn -0.442863 -0.238856 -0.864188  
vn -0.309017 -0.500000 -0.809017  
vn -0.162460 -0.262866 -0.951056  
vn  0.000000 -0.850651 -0.525731  
vn -0.147621 -0.716567 -0.681718  
vn  0.147621 -0.716567 -0.681718  
vn  0.000000 -0.525731 -0.850651  
vn  0.309017 -0.500000 -0.809017  
vn  0.442863 -0.238856 -0.864188  
vn  0.162460 -0.262866 -0.951056  
vn  0.238856 -0.864188 -0.442863  
vn  0.500000 -0.809017 -0.309017  
vn  0.425325 -0.688191 -0.587785  
vn  0.716567 -0.681718 -0.147621  
vn  0.688191 -0.587785 -0.425325  
vn  0.587785 -0.425325 -0.688191  
vn  0.000000 -0.955423 -0.295242  
vn  0.000000 -1.000000  0.000000  
vn  0.262866 -0.951056 -0.162460  
vn  0.000000 -0.850651  0.525731  
vn  0.000000 -0.955423  0.295242  
vn  0.238856 -0.864188  0.442863  
vn  0.262866 -0.951056  0.162460  
vn  0.500000 -0.809017  0.309017  
vn  0.716567 -0.681718  0.147621  
vn  0.525731 -0.850651  0.000000  
vn -0.238856 -0.864188 -0.442863  
vn -0.500000 -0.809017 -0.309017  
vn -0.262866 -0.951056 -0.162460  
vn -0.850651 -0.525731  0.000000  
vn -0.716567 -0.681718 -0.147621  
vn -0.716567 -0.681718  0.147621  
vn -0.525731 -0.850651  0.000000  
vn -0.500000 -0.809017  0.309017  
vn -0.238856 -0.864188  0.442863  
vn -0.262866 -0.951056  0.162460  
vn -0.864188 -0.442863  0.238856  
vn -0.809017 -0.309017  0.500000  
vn -0.688191 -0.587785  0.425325  
vn -0.681718 -0.147621  0.716567  
vn -0.442863 -0.238856  0.864188  
vn -0.587785 -0.425325  0.688191  
vn -0.309017 -0.500000  0.809017  
vn -0.147621 -0.716567  0.681718  
vn -0.425325 -0.688191  0.587785  
vn -0.162460 -0.262866  0.951056  
vn  0.442863 -0.238856  0.864188  
vn  0.162460 -0.262866  0.951056  
vn  0.309017 -0.500000  0.809017  
vn  0.147621 -0.716567  0.681718  
vn  0.000000 -0.525731  0.850651  
vn  0.425325 -0.688191  0.587785  
vn  0.587785 -0.425325  0.688191  
vn  0.688191 -0.587785  0.425325  
vn -0.955423  0.295242  0.000000  
vn -0.951056  0.162460  0.262866  
vn -1.000000  0.000000  0.000000  
vn -0.850651  0.000000  0.525731  
vn -0.955423 -0.295242  0.000000  
vn -0.951056 -0.162460  0.262866  
vn -0.864188  0.442863 -0.238856  
vn -0.951056  0.162460 -0.262866  
vn -0.809017  0.309017 -0.500000  
vn -0.864188 -0.442863 -0.238856  
vn -0.951056 -0.162460 -0.262866  
vn -0.809017 -0.309017 -0.500000  
vn -0.681718  0.147621 -0.716567  
vn -0.681718 -0.147621 -0.716567  
vn -0.850651  0.000000 -0.525731  
vn -0.688191  0.587785 -0.425325  
vn -0.587785  0.425325 -0.688191  
vn -0.425325  0.688191 -0.587785  
vn -0.425325 -0.688191 -0.587785  
vn -0.587785 -0.425325 -0.688191  
vn -0.688191 -0.587785 -0.425325 
{{range .VT}}vt {{ index . 0 }} {{ index . 1 }}
{{end}}{{range .F}}f {{ index . 0 }}/{{ index . 1 }}/{{ index . 2 }} {{ index . 3 }}/{{ index . 4 }}/{{ index . 5 }} {{ index . 6 }}/{{ index . 7 }}/{{ index . 8 }}
{{end}}`

const mtl = `newmtl {{ .Name }}
Ka 1.000000 1.000000 1.000000
Kd 1.000000 1.000000 1.000000
Ks 0.000000 0.000000 0.000000
Tr 1.000000
illum 1
Ns 0.000000
map_Kd {{ .Name }}.jpg
`
