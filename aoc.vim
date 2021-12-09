if exists("b:current_syntax")
  finish
endif

syn match aocComment  "#.*$"
syn match aocLabel    "[a-z][a-z0-9_]*:"
syn region aocString  start="'" end="'"

syn keyword aocKw fn
syn keyword aocKw if
syn keyword aocKw return

syn keyword aocFn print
syn keyword aocFn push
syn keyword aocFn len
syn keyword aocFn split

hi def link aocComment  Comment
hi def link aocLabel    Label
hi def link aocString   String
hi def link aocKw       Keyword
hi def link aocFn       Identifier
