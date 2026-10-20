class HelloWorld
  def draw()
    clear()
    text(1, 6, "Hello", 0x00FF88)
    text(1, 14, "World", 0xFFFFFF)
  end
end

return HelloWorld()
