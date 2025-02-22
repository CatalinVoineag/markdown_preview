package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

const (
  defaultTemplate = `<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8">
    <title>{{ .Title }}</title>
  </head>
  <body>
{{ .Body }}
  </body>
</html>
`
)

type content struct {
  Title string
  Body template.HTML
}

func main() {
  // parse flags
  filename := flag.String("file", "", "Markdown file to preview")
  skipPreview := flag.Bool("s", false, "Skip auto-preview")
  tFname := flag.String("t", "", "Alternate template name")
  flag.Parse()

  // If user did not proivde input file, show usage
  if *filename == "" {
    flag.Usage()
    os.Exit(1)
  }

  if err := run(*filename, *tFname, os.Stdout, *skipPreview); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

func run(filename string, tFname string, out io.Writer, skipPreview bool) error {
  // READ all the data from the input file and check for errors
  input, err := os.ReadFile(filename)
  if err != nil {
    return err
  }
  htmlData, err := parseContent(input, tFname)
  if err != nil {
    return err
  }
  // Create temp file and check errors
  temp, err := os.CreateTemp("", "markdown_preview*.html")
  if err != nil {
    return err
  }
  if err := temp.Close(); err != nil {
    return err
  }
  outName := temp.Name()
  fmt.Fprintln(out, outName)


  if err := saveHTML(outName, htmlData); err != nil {
    return err
  }

  if skipPreview {
    return nil
  }
  defer os.Remove(outName)

  return preview(outName)
}

func parseContent(input []byte, tFname string) ([]byte, error) {
  //Parse the markdown file through blackfriday and bluemonday
  // to generate a valid and safe HTML
  output := blackfriday.Run(input)
  body := bluemonday.UGCPolicy().SanitizeBytes(output)

  // Parse the contents of the defaultTemplate into a new Template
  t, err := template.New("mdp").Parse(defaultTemplate)
  if err != nil {
    return nil, err
  }

  // If user provided alternate tempalte, replace template
  if tFname != "" {
    t, err = template.ParseFiles(tFname)
    if err != nil {
      return nil, err
    }
  }

  // Instantiate content, adding title and body
  c := content{
    Title: "Markdown Preview Tool",
    Body: template.HTML(body),
  }

  //Create a buffer of bytes to write to file
  var buffer bytes.Buffer

  // Execute the template with the content type
  if err := t.Execute(&buffer, c); err != nil {
    return nil, err
  }

  return buffer.Bytes(), nil
}

func saveHTML(outFname string, data []byte) error {
  // Write the bytes to the file
  return os.WriteFile(outFname, data, 0644)
}

func preview(fname string) error {
  cName := ""
  cParams := []string{}

  // Define executable based on OS
  switch runtime.GOOS {
  case "linux":
    cName = "xdg-open"
  case "windows":
    cName = "cmd.exe"
    cParams = []string{"/C", "start"}
  case "darwin":
    cName = "open"
  default: 
    return fmt.Errorf("OS not supported")
  }

  // Append filename to param slice
  cParams = append(cParams, fname)
  // Locate executable in PATH
  cPath, err := exec.LookPath(cName)

  if err != nil {
    return err
  }

  // Open the file using default program
  err = exec.Command(cPath, cParams...).Run()

  // Give the browser time to owpen file before deleting it
  time.Sleep(2 * time.Second)
  return err;
}
