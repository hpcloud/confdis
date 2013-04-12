require 'redis'
require 'json'

# TODO: write an async version using eventmachine.
class ConfDis
  attr_reader :data
  
  def initialize(host, port, rootkey)
    @host = host
    @port = port
    @rootkey = rootkey
    @pubchannel = "#{rootkey}:_changes"
    @redis = Redis.new :host => host, :port => port
    reload
    puts @data
  end

  def stop
    @redis.disconnect
    @redis = nil
  end
  
  def reload
    puts "reloading."
    @data = JSON.parse(@redis.get(@rootkey) || '{}')
  end

  # save in-memory config data back to redis (non-atomic).
  def save
    @redis.set(@rootkey, @data.to_json)
    @redis.publish @pubchannel, true
  end

  # Watch for changes from other clients, and update @data accordingly.
  def watch
    # FIXME: writing to @data (via `reload`) from multiple threads is
    # not safe. switch to using EM.
    Thread.new do
      @redis.subscribe(@pubch) do |on|
        on.subscribe do |ch, subs|
          puts "subscribed to #{ch}"
        end
        on.message do |ch, msg|
          puts "got msg"
          reload
        end
      end
    end
  end
end
