why not write a programming language to write Advent of Code 2021 solutions in?

a very good idea.

a hacky tree-walk interpreter in go.

a program is bunch of sections. a section is a label, followed by a colon, followed by a block or an expression. a block is a bunch of statements in curly brackets. an expression is what you'd expect.

```
file: 'input.txt'

part1: {
  print(input)
  print(lines)

  var sum = 0
  for line in lines {
    sum = sum + num(line)
  }

  print(sum)
  print('do stuff here')
}

part2: {
  print('do other stuff here')
}
```

the interpreter will do the following:

  1. evaluate the file section to find the filename
  2. read the file
  3. evaluate the part1 section
  4. evaluate the part2 section

the input file is available in the `input` variable. the input, split into lines, is availiable in the `lines` object. objects are key/value maps. there are no arrays.
