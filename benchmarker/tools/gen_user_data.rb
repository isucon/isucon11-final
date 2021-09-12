require 'forgery_ja'
require 'securerandom'
require 'bcrypt'
require 'ulid'

srand(100)

ForgeryJa.dictionaries.reset!
ForgeryJa.load_paths << __dir__

def gen_user_data(code_min, code_max, is_teacher)
  (code_min..code_max).map { |i|
    first_name = ForgeryJa(:name).first_name(to: ForgeryJa::ARRAY)
    last_name = ForgeryJa(:name).last_name(to: ForgeryJa::ARRAY)

    id = ULID.generate
    code = sprintf("%s%05d", is_teacher ? "T" : "S", i)
    full_name = last_name[ForgeryJa::KANJI] + " " + first_name[ForgeryJa::KANJI]
    password = SecureRandom.alphanumeric(10)

    { id: id, code: code, full_name: full_name, password: password }
  }
end

def save_tsv(users, file_name)
  File.open(file_name, mode = "w") do |f|
    users.each do |user|
      f.write(user[:id], "\t", user[:code], "\t", user[:full_name], "\t", user[:password], "\n")
    end
  end
end

def save_sql(users, file_name, is_teacher)
  File.open(file_name, mode = "a") do |f|
    f.write("INSERT INTO `users` (`id`, `code`, `name`, `hashed_password`, `type`) VALUES\n")
    s = users.map{|user|
      sprintf(
        "('%s','%s','%s','%s','%s')",
        user[:id],
        user[:code],
        user[:full_name],
        BCrypt::Password.create(user[:password], :cost => 4),
        is_teacher ? "teacher" : "student"
      )
    }.join(",\n")
    f.write(s, ";\n")
  end
end

if File.exist?("init.sql")
  File.delete("init.sql")
end

teachers = [{ id: ULID.generate, code: "T00000", full_name: "isucon(教員)", password: "isucon" }]
teachers.concat(gen_user_data(1, 49, true))
students = [{ id: ULID.generate, code: "S00000", full_name: "isucon(学生)", password: "isucon" }]
students.concat(gen_user_data(1, 4999, false))

save_tsv(teachers[1..], "teacher.tsv")
save_tsv(students[1..], "student.tsv")
save_sql(teachers, "init.sql", true)
save_sql(students, "init.sql", false)
