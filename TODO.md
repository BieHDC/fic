## fixme
- grep for the various 'fixme's
- nothing has tests
- find a nice person to make a proper icon

---
## other fixme
- storage seems to handle differently depending on a terminating slash
- - workarounding with parentfromfile()

---
## todo
- when speed constrained, ensure that every image was shown once in the player, no skipping  
- - does fyne have a way to hook on swap? aka syncronise  
- - i have only found "func (c *glCanvas) Capture() image.Image" to hook into the draw thread  
- support more formats  
- - webm (pain https://github.com/at-wat/ebml-go)  
- - - maybe https://github.com/metal3d/fyne-streamer would be a good alternative, but very linux-y only  
- - animated webp (static works)  

---
## things to explore
- support actions on filecards  
- - right click context menu might make most sense (copy path, open path, ...expandable)  
- - figure out what options we really want and need  
- - same for the displayed images  
- the whole playerconstruct does not like zero len things (like empty folders)  
- - is it worth special casing this, or do we depend on filetreemap to never add empty folders?  
- play does not block and therefore skips quickily to the next valid one  
- - should the previous and next buttons act the same, jumping to the next valid image?  
