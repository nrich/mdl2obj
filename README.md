# mdl2obj

Converts Quake1 model files to Wavefront OBJ model data. It will only
convert the first frame of a model (in case the model is animated).
It does not extract model texture data.

## Usage

    mdl2obj <model.mdl>

It will read the input model file and will create the following
files in the current working directory:

    model.obj
    model.mtl

The model's material `model.mtl` file expects a texture file named
`model.jpg` in the same directory.

The name `model` for all file names above is examplary. All names change according to the basename of the input model file name.

## Build

    go build mdl2obj.go
