if exists("b:current_syntax")
  finish
endif

syn match aocComment  "#.*$"
syn match aocLabel    "[a-z][a-z0-9_]*:"
syn region aocString  start="'" end="'"

syn keyword aocKw fn
syn keyword aocKw if
syn keyword aocKw else
syn keyword aocKw return
syn keyword aocKw for
syn keyword aocKw in
syn keyword aocKw var
syn keyword aocKw continue
syn keyword aocKw break
syn keyword aocKw match

syn keyword aocFn print
syn keyword aocFn push
syn keyword aocFn delete
syn keyword aocFn len
syn keyword aocFn split
syn keyword aocFn read
syn keyword aocFn num

hi def link aocComment  Comment
hi def link aocLabel    Label
hi def link aocString   String
hi def link aocKw       Keyword
hi def link aocFn       Identifier
