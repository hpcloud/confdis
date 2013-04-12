require 'helper'

class TestConfdis < Test::Unit::TestCase
  should "connect properly" do
    # TODO: test watcher
    name = "Primates - #{Time.now}"
    c = ConfDis.new "localhost", 6379, "test:confdis:simple"
    c2 = ConfDis.new "localhost", 6379, "test:confdis:simple"
    c.data[:name] = name
    c.save
    sleep 1
    # assert_equal name, c2.data[:name]
  end
end
