module Isucholar
  module Util
    def self.get_env(key, val)
      retval = ENV.fetch(key, '')
      return val if retval.empty?
      retval
    end

    def self.average(arr, or_)
      return or_ if arr.empty?
      sum = arr.sum
      sum.to_f / arr.size
    end

    def self.max(arr, or_)
      return or_ if arr.empty?
      arr.max
    end

    def self.min(arr, or_)
      return or_ if arr.empty?
      arr.min
    end

    def self.stddev(arr, avg)
      return 0.0 if arr.empty?
      sdm_sum = arr.inject(0.0) { |r,i| r + ( (i.to_f-avg) ** 2 ) }
      Math.sqrt(sdm_sum / arr.size)
    end

    def self.t_score(v, arr)
      avg = average(arr, 0)
      stddev = stddev(arr, avg)
      if stddev == 0
        50
      else
        (v.to_f - avg) / stddev * 10 + 50
      end
    end

    def self.all_equal?(arr)
      arr.uniq.size == 1
    end
  end
end
