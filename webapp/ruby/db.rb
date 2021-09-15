require 'mysql2'
require 'mysql2-cs-bind'

require_relative './util'

module Isucholar
  module DB
    def self.get_db(batch = nil)
      (Thread.current[:isucholar_db] ||= [])[batch ? 1 : 0] ||= Mysql2::Client.new(
        host: Util.get_env('MYSQL_HOSTNAME', '127.0.0.1'),
        port: Util.get_env('MYSQL_PORT', 3306).to_i,
        username: Util.get_env('MYSQL_USER', 'isucon'),
        password: Util.get_env('MYSQL_PASS', 'isucon'),
        database: Util.get_env('MYSQL_DATABASE', 'isucholar'),
        charset: 'utf8mb4',
        database_timezone: :utc,
        cast_booleans: true,
        symbolize_keys: true,
        reconnect: true,
        init_command: "SET time_zone='+00:00';",
        flags: batch ? Mysql2::Client::MULTI_STATEMENTS : nil,
      )
    end
  end
end
