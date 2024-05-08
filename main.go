package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	// "strings"

	sitter "github.com/garid3000/go-tree-sitter" //sitter "github.com/smacker/go-tree-sitter"
	"github.com/garid3000/go-tree-sitter/org"

	"hash/maphash"
	"time"
)

var org_html_toplevel_hlevel = 2   // org-html-toplevel-hlevel  currently Only tested on 2
var fileString []byte              // this contains the source code
var outputFile *os.File            // this for the ooutput file
var treeNodeId = make([]int, 100)  // this contains the id
var orgHeaderId = make([]int, 100) // this contains the id
var hmap maphash.Hash

var (
	inputOrgFilePath string
	outputPath       string
	outputType       string // html,dir (tbd: later may txt,md,graph?)
	outputMidProcess string
	// later add org-roam.db path
)

// var map_of_nodes map[string]string // 1-1-2 -> uint64
func slice_of_int_to_string(alist []int, depht int, sep string, endWithSep bool) string {
	result := ""
	for i, element := range alist {
		if i >= depht {
			if i != 0 && endWithSep == false {
				result = result[:len(result)-1] // just to make sure it won't end with -
			}
			break
		}
		result += fmt.Sprintf("%d%s", element, sep)
	}
	return result
}

func slice_of_int_to_hash_string(alist []int, depht int) string {
	hmap.SetSeed(hmap.Seed())
	hmap.WriteString(slice_of_int_to_string(alist, depht, "-", false)) // should have error handler?
	return "org" + strconv.FormatUint(hmap.Sum64(), 36)
}

func getPreChildHTML(node *sitter.Node, orgHeaderDepth int) (result string) {
	// headerDepth represents number of stars

	switch node.Type() {
	case "headline":
		if orgHeaderDepth <= 3 { // check this 3 later, probably related with org_html_toplevel_hlevel
			result = fmt.Sprintf(
				"\n<h%d id=\"%s\">",
				orgHeaderDepth-1+org_html_toplevel_hlevel,
				slice_of_int_to_hash_string(orgHeaderId, orgHeaderDepth),
			)
		} else {
			result = fmt.Sprintf(
				//"\n<h%d id=\"%s\">",
				"<a id=\"%s\"></a>", //asdflkjasdf (4 star)<br />
				// orgHeaderDepth-1+org_html_toplevel_hlevel,
				slice_of_int_to_hash_string(orgHeaderId, orgHeaderDepth),
			)
		}
	case "section":
		if orgHeaderDepth <= 3 {
			result = fmt.Sprintf(
				"\n\n<div id=\"outline-container-%s\" class=\"outline-%d\">",
				slice_of_int_to_hash_string(orgHeaderId, orgHeaderDepth),
				orgHeaderDepth-1+org_html_toplevel_hlevel,
			)
		} else {
			result = "\n<li>"
		}

	case "stars": // probably child of header
		if orgHeaderDepth <= 3 { // check this 3 later, probably related with org_html_toplevel_hlevel
			result = fmt.Sprintf(
				"<span class=\"section-number-%d\">%s",
				orgHeaderDepth-1+org_html_toplevel_hlevel,
				slice_of_int_to_string(orgHeaderId, orgHeaderDepth, ".", true),
			)
		}
	case "item":
		result = " " + node.Content(fileString) // TODO

	case "body":
		// apparently org does it when there is body of text without any headers at the begining
		if orgHeaderDepth-1+org_html_toplevel_hlevel == 1 {
			result = ""
		} else {
			result = fmt.Sprintf(
				"\n<div class=\"outline-text-%d\" id=\"text-%s\">",
				orgHeaderDepth-1+org_html_toplevel_hlevel,
				slice_of_int_to_string(orgHeaderId, orgHeaderDepth, "-", false),
			)
		}

	case "paragraph":
		// if node.Parent().Type() == "list/listitem" then none?
		result = fmt.Sprintf("\n<p>\n%s", node.Content(fileString))

	case "list":
		// need to ordered or unordered list

		//if node.ChildByFieldName("listitem").ChildByFieldName("bullet").Content(fileString) == "1." {
		if node.Child(0).Child(0).Content(fileString) == "1." {
			result = "\n<ol class=\"org-ol\">"
		} else {
			result = "\n<ul class=\"org-ul\">"
		}

	case "listitem":
		// need to ordered or unordered list
		result = "\n<li>"

	default:
	}
	return result
}

func getPostChildHTML(node *sitter.Node, orgHeaderDepth int) (result string) {
	switch node.Type() {
	case "headline":
		if orgHeaderDepth <= 3 {
			result = fmt.Sprintf("</h%d>", orgHeaderDepth-1+org_html_toplevel_hlevel)
		} else {
			result = "<br />"
		}
	case "section":
		if orgHeaderDepth <= 3 {
			result = "\n</div>"
		} else {
			result = "\n</li>" //\n</ol>
		}
	case "body":
		if orgHeaderDepth-1+org_html_toplevel_hlevel == 1 {
			result = ""
		} else {
			result = "\n</div>"
		}
	case "item":
		result = ""
	case "stars":
		if orgHeaderDepth <= 3 { // check this 3 later, probably related with org_html_toplevel_hlevel
			result = "</span>"
		}
	case "paragraph":
		result = "</p>"

	case "list":
		// need to ordered or unordered list
		//fmt.Printf(node.Content(fileString))
		if node.Child(0).Child(0).Content(fileString) == "1." {
			result = "\n</ol>"
		} else {
			result = "\n</ul>"
		}

	case "listitem":
		// need to ordered or unordered list
		result = "\n</li>"

	default:
	}
	return result
}

// returns the count of child with specific type.
func CountChildByType(node *sitter.Node, childType string) (result int) {
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == childType{
			result++
		}
	}
	return result
}

func SquirrelJumpToTreeNodeForHTML(node *sitter.Node, treeDepth int, orgHeaderDepth int, outputfile *os.File) {
	//thing before visiting child-nodes
	outputfile.WriteString(getPreChildHTML(node, orgHeaderDepth))

	//visit the child-nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child_node := node.Child(i)
		treeNodeId[treeDepth] = i // should be the section
		treeNodeId[treeDepth+1] = 0
		if child_node.Type() == "section" {
			orgHeaderId[orgHeaderDepth]++     // should start at 0 and increment (it should get 0 on the previous/parent recursion)
			orgHeaderId[orgHeaderDepth+1] = 0 //
			// headline, body, section
			// next_depth+ // depth changes only when readling child of section?
			// i needed something just before/after all-child-sections after depht of 3
			if orgHeaderId[orgHeaderDepth] == 1 && orgHeaderDepth > 3-1  { // when it's first child-section && CountChildByType(node, "section") != 1
				outputfile.WriteString("\n\n<ol class=\"org-ol\">")
			}
			SquirrelJumpToTreeNodeForHTML(child_node, treeDepth+1, orgHeaderDepth+1, outputfile)

			if i == int(node.ChildCount())-1 && orgHeaderDepth > 3 -1 { // when it's last child section (TODO) && CountChildByType(node, "section") != 1
				outputfile.WriteString("\n\n</ol>")
			}
		} else {
			SquirrelJumpToTreeNodeForHTML(child_node, treeDepth+1, orgHeaderDepth, outputfile)
		}
	}

	//thing after visiting child-nodes
	outputfile.WriteString(getPostChildHTML(node, orgHeaderDepth))
}

func PreSquirrelJumpToTreeNodeForHTML(outputfile *os.File, now time.Time) {
	outputfile.WriteString(
		`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN"
"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="en" xml:lang="en">
<head>
` + fmt.Sprintf("<!-- %d-%02d-%02d xxx %02d:%02d -->", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()) +
			`
<meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>&lrm;</title>
<meta name="generator" content="Org Mode" />
<style>
  #content { max-width: 60em; margin: auto; }
  .title  { text-align: center;
             margin-bottom: .2em; }
  .subtitle { text-align: center;
              font-size: medium;
              font-weight: bold;
              margin-top:0; }
  .todo   { font-family: monospace; color: red; }
  .done   { font-family: monospace; color: green; }
  .priority { font-family: monospace; color: orange; }
  .tag    { background-color: #eee; font-family: monospace;
            padding: 2px; font-size: 80%; font-weight: normal; }
  .timestamp { color: #bebebe; }
  .timestamp-kwd { color: #5f9ea0; }
  .org-right  { margin-left: auto; margin-right: 0px;  text-align: right; }
  .org-left   { margin-left: 0px;  margin-right: auto; text-align: left; }
  .org-center { margin-left: auto; margin-right: auto; text-align: center; }
  .underline { text-decoration: underline; }
  #postamble p, #preamble p { font-size: 90%; margin: .2em; }
  p.verse { margin-left: 3%; }
  pre {
    border: 1px solid #e6e6e6;
    border-radius: 3px;
    background-color: #f2f2f2;
    padding: 8pt;
    font-family: monospace;
    overflow: auto;
    margin: 1.2em;
  }
  pre.src {
    position: relative;
    overflow: auto;
  }
  pre.src:before {
    display: none;
    position: absolute;
    top: -8px;
    right: 12px;
    padding: 3px;
    color: #555;
    background-color: #f2f2f299;
  }
  pre.src:hover:before { display: inline; margin-top: 14px;}
  /* Languages per Org manual */
  pre.src-asymptote:before { content: 'Asymptote'; }
  pre.src-awk:before { content: 'Awk'; }
  pre.src-authinfo::before { content: 'Authinfo'; }
  pre.src-C:before { content: 'C'; }
  /* pre.src-C++ doesn't work in CSS */
  pre.src-clojure:before { content: 'Clojure'; }
  pre.src-css:before { content: 'CSS'; }
  pre.src-D:before { content: 'D'; }
  pre.src-ditaa:before { content: 'ditaa'; }
  pre.src-dot:before { content: 'Graphviz'; }
  pre.src-calc:before { content: 'Emacs Calc'; }
  pre.src-emacs-lisp:before { content: 'Emacs Lisp'; }
  pre.src-fortran:before { content: 'Fortran'; }
  pre.src-gnuplot:before { content: 'gnuplot'; }
  pre.src-haskell:before { content: 'Haskell'; }
  pre.src-hledger:before { content: 'hledger'; }
  pre.src-java:before { content: 'Java'; }
  pre.src-js:before { content: 'Javascript'; }
  pre.src-latex:before { content: 'LaTeX'; }
  pre.src-ledger:before { content: 'Ledger'; }
  pre.src-lisp:before { content: 'Lisp'; }
  pre.src-lilypond:before { content: 'Lilypond'; }
  pre.src-lua:before { content: 'Lua'; }
  pre.src-matlab:before { content: 'MATLAB'; }
  pre.src-mscgen:before { content: 'Mscgen'; }
  pre.src-ocaml:before { content: 'Objective Caml'; }
  pre.src-octave:before { content: 'Octave'; }
  pre.src-org:before { content: 'Org mode'; }
  pre.src-oz:before { content: 'OZ'; }
  pre.src-plantuml:before { content: 'Plantuml'; }
  pre.src-processing:before { content: 'Processing.js'; }
  pre.src-python:before { content: 'Python'; }
  pre.src-R:before { content: 'R'; }
  pre.src-ruby:before { content: 'Ruby'; }
  pre.src-sass:before { content: 'Sass'; }
  pre.src-scheme:before { content: 'Scheme'; }
  pre.src-screen:before { content: 'Gnu Screen'; }
  pre.src-sed:before { content: 'Sed'; }
  pre.src-sh:before { content: 'shell'; }
  pre.src-sql:before { content: 'SQL'; }
  pre.src-sqlite:before { content: 'SQLite'; }
  /* additional languages in org.el's org-babel-load-languages alist */
  pre.src-forth:before { content: 'Forth'; }
  pre.src-io:before { content: 'IO'; }
  pre.src-J:before { content: 'J'; }
  pre.src-makefile:before { content: 'Makefile'; }
  pre.src-maxima:before { content: 'Maxima'; }
  pre.src-perl:before { content: 'Perl'; }
  pre.src-picolisp:before { content: 'Pico Lisp'; }
  pre.src-scala:before { content: 'Scala'; }
  pre.src-shell:before { content: 'Shell Script'; }
  pre.src-ebnf2ps:before { content: 'ebfn2ps'; }
  /* additional language identifiers per "defun org-babel-execute"
       in ob-*.el */
  pre.src-cpp:before  { content: 'C++'; }
  pre.src-abc:before  { content: 'ABC'; }
  pre.src-coq:before  { content: 'Coq'; }
  pre.src-groovy:before  { content: 'Groovy'; }
  /* additional language identifiers from org-babel-shell-names in
     ob-shell.el: ob-shell is the only babel language using a lambda to put
     the execution function name together. */
  pre.src-bash:before  { content: 'bash'; }
  pre.src-csh:before  { content: 'csh'; }
  pre.src-ash:before  { content: 'ash'; }
  pre.src-dash:before  { content: 'dash'; }
  pre.src-ksh:before  { content: 'ksh'; }
  pre.src-mksh:before  { content: 'mksh'; }
  pre.src-posh:before  { content: 'posh'; }
  /* Additional Emacs modes also supported by the LaTeX listings package */
  pre.src-ada:before { content: 'Ada'; }
  pre.src-asm:before { content: 'Assembler'; }
  pre.src-caml:before { content: 'Caml'; }
  pre.src-delphi:before { content: 'Delphi'; }
  pre.src-html:before { content: 'HTML'; }
  pre.src-idl:before { content: 'IDL'; }
  pre.src-mercury:before { content: 'Mercury'; }
  pre.src-metapost:before { content: 'MetaPost'; }
  pre.src-modula-2:before { content: 'Modula-2'; }
  pre.src-pascal:before { content: 'Pascal'; }
  pre.src-ps:before { content: 'PostScript'; }
  pre.src-prolog:before { content: 'Prolog'; }
  pre.src-simula:before { content: 'Simula'; }
  pre.src-tcl:before { content: 'tcl'; }
  pre.src-tex:before { content: 'TeX'; }
  pre.src-plain-tex:before { content: 'Plain TeX'; }
  pre.src-verilog:before { content: 'Verilog'; }
  pre.src-vhdl:before { content: 'VHDL'; }
  pre.src-xml:before { content: 'XML'; }
  pre.src-nxml:before { content: 'XML'; }
  /* add a generic configuration mode; LaTeX export needs an additional
     (add-to-list 'org-latex-listings-langs '(conf " ")) in .emacs */
  pre.src-conf:before { content: 'Configuration File'; }

  table { border-collapse:collapse; }
  caption.t-above { caption-side: top; }
  caption.t-bottom { caption-side: bottom; }
  td, th { vertical-align:top;  }
  th.org-right  { text-align: center;  }
  th.org-left   { text-align: center;   }
  th.org-center { text-align: center; }
  td.org-right  { text-align: right;  }
  td.org-left   { text-align: left;   }
  td.org-center { text-align: center; }
  dt { font-weight: bold; }
  .footpara { display: inline; }
  .footdef  { margin-bottom: 1em; }
  .figure { padding: 1em; }
  .figure p { text-align: center; }
  .equation-container {
    display: table;
    text-align: center;
    width: 100%;
  }
  .equation {
    vertical-align: middle;
  }
  .equation-label {
    display: table-cell;
    text-align: right;
    vertical-align: middle;
  }
  .inlinetask {
    padding: 10px;
    border: 2px solid gray;
    margin: 10px;
    background: #ffffcc;
  }
  #org-div-home-and-up
   { text-align: right; font-size: 70%; white-space: nowrap; }
  textarea { overflow-x: auto; }
  .linenr { font-size: smaller }
  .code-highlighted { background-color: #ffff00; }
  .org-info-js_info-navigation { border-style: none; }
  #org-info-js_console-label
    { font-size: 10px; font-weight: bold; white-space: nowrap; }
  .org-info-js_search-highlight
    { background-color: #ffff00; color: #000000; font-weight: bold; }
  .org-svg { }
</style>
</head>
<body>
`,
	)
}

func ContentsSquirrelJumpToTreeNodeForHTML(node *sitter.Node, orgHeaderDepth int, outputfile *os.File) {
	outputfile.WriteString( // contents prelude
		`<div id="content" class="content">
<div id="table-of-contents" role="doc-toc">
<h2>Table of Contents</h2>
<div id="text-table-of-contents" role="doc-toc">`)
	getTableOfContentsHTML(node, 0, outputfile)
	clear(orgHeaderId)

	outputfile.WriteString( // contents postlude
		`
</div>
</div>
`)

}

func PostSquirrelJumpToTreeNodeForHTML(outputfile *os.File, now time.Time) {
	outputfile.WriteString(
		`
</div>
<div id="postamble" class="status">
` + fmt.Sprintf("<p class=\"date\">Created: %d-%02d-%02d XXX %02d:%02d</p>", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()) + `
<p class="validation"><a href="https://validator.w3.org/check?uri=referer">Validate</a></p>
</div>
</body>
</html>
`,
	)
}

func doesNodeHasChildNamedThis(node *sitter.Node, child_type string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == child_type {
			return true
		}
	}
	return false
}

func getTableOfContentsHTML(node *sitter.Node, orgHeaderDepth int, outputfile *os.File) {
	switch node.Type() {
	case "section":
		outputfile.WriteString(
			fmt.Sprintf(
				"\n<li><a href=\"#%s\">%s %s</a>",
				slice_of_int_to_hash_string(orgHeaderId, orgHeaderDepth),
				slice_of_int_to_string(orgHeaderId, orgHeaderDepth, ".", true),
				node.ChildByFieldName("headline").ChildByFieldName("item").Content(fileString),
			),
		)
	}

	nodeHasChildSection := doesNodeHasChildNamedThis(node, "section")
	if nodeHasChildSection {
		outputfile.WriteString("\n<ul>")
		for i := 0; i < int(node.ChildCount()); i++ {
			child_node := node.Child(i)

			if child_node.Type() == "section" {
				orgHeaderId[orgHeaderDepth]++     // should start at 0 and increment (it should get 0 on the previous/parent recursion)
				orgHeaderId[orgHeaderDepth+1] = 0 //

				getTableOfContentsHTML(child_node, orgHeaderDepth+1, outputfile)
			} else {
				//getTableOfContentsHTML(child_node, orgHeaderDepth, outputfile)
				// I dont have to enter child that's not a section (headers or body of section)
			}
		}
		outputfile.WriteString("\n</ul>")
	}

	switch node.Type() {
	case "section":
		if nodeHasChildSection {
			outputfile.WriteString(fmt.Sprintf("\n"))
		}
		outputfile.WriteString(fmt.Sprintf("</li>"))
	}

}

func main() {
	// reaind cmd flags
	flag.StringVar(&inputOrgFilePath, "in", "README.org", "a org file")
	flag.StringVar(&outputPath, "out", "README.html", "a output file or dir path")
	flag.StringVar(&outputType, "type", "html", "the file path")
	flag.Parse()

	var err error
	fileString, err = os.ReadFile(inputOrgFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// parse the tree
	lang := org.GetLanguage()
	rootNode, err := sitter.ParseCtx(context.Background(), fileString, lang)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	if outputType == "html" {
		outputfile, err := os.Create(outputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer outputfile.Close()

		//surfing the nodes
		PreSquirrelJumpToTreeNodeForHTML(outputfile, now)
		ContentsSquirrelJumpToTreeNodeForHTML(rootNode, 0, outputfile)
		SquirrelJumpToTreeNodeForHTML(rootNode, 0, 0, outputfile)
		PostSquirrelJumpToTreeNodeForHTML(outputfile, now)
	} else if outputType == "dir" {

	}
}
