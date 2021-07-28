require 'forgery_ja'
require 'securerandom'
require 'bcrypt'

ForgeryJa.dictionaries.reset!
ForgeryJa.load_paths << __dir__

def gen_user_data(max, is_teacher)
  users = []
  (0..max-1).each { |i|
    first_name = ForgeryJa(:name).first_name(to: ForgeryJa::ARRAY)
    last_name = ForgeryJa(:name).last_name(to: ForgeryJa::ARRAY)

    uuid = SecureRandom.uuid
    code = sprintf("%s%05d", is_teacher ? "T" : "S", i)
    full_name = last_name[ForgeryJa::KANJI] + " " + first_name[ForgeryJa::KANJI]
    password = SecureRandom.alphanumeric(10)

    users.push({ uuid: uuid, code: code, full_name: full_name, password: password })
  }
  users
end

def save_tsv(users, file_name)
  File.open(file_name, mode = "w") do |f|
    users.each do |user|
      f.write(user[:code], "\t", user[:full_name], "\t", user[:password], "\n")
    end
  end
end

def save_sql(users, file_name, is_teacher)
  File.open(file_name, mode = "a") do |f|
    f.write("INSERT INTO `users` (`id`, `code`, `name`, `hashed_password`, `type`) VALUES\n")
    s = users.map{|user|
      sprintf(
        "('%s','%s','%s','%s','%s')",
        user[:uuid],
        user[:code],
        user[:full_name],
        BCrypt::Password.create(user[:password], :cost => 10),
        is_teacher ? "teacher" : "student"
      )
    }.join(",\n")
    f.write(s, ";\n")
  end
end

if File.exist?("init.sql")
  File.delete("init.sql")
end

teachers = gen_user_data(50, true)
save_tsv(teachers, "faculty.tsv")
save_sql(teachers, "init.sql", true)

students = gen_user_data(5000, false)
save_tsv(students, "student.tsv")
save_sql(students, "init.sql", false)
