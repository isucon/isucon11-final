require 'bcrypt'
require 'mysql2'
require 'mysql2-cs-bind'
require 'net/http'
require 'sinatra/base'
require 'ulid'
require 'uri'

require_relative './util'
require_relative './db'


module Isucholar
  class App < Sinatra::Base
    configure :development do
      require 'sinatra/reloader'
      register Sinatra::Reloader
    end

    SQL_DIRECTORY = '../sql/'
    ASSIGNMENTS_DIRECTORY = '../assignments/'
    INIT_DATA_DIRECTORY = '../data/'
    SESSION_NAME = 'isucholar_ruby'
    MYSQL_ERR_NUM_DUPLICATE_ENTRY = 1062

    STUDENT = "student"
    TEACHER = "teacher"

    STATUS_REGISTRATION = "registration"
    STATUS_IN_PROGRESS = "in-progress"
    STATUS_CLOSED = "closed"

    LIBERAL_ARTS = "liberal-arts"
    MAJOR_SUBJECTS = "major-subjects"

    MONDAY = "monday"
    TUESDAY = "tuesday"
    WEDNESDAY = "wednesday"
    THURSDAY = "thursday"
    FRIDAY = "friday"

    set :session_secret, 'trapnomura'
    set :sessions, key: SESSION_NAME
    set :protection, false

    set :public_folder, '../frontend/dist/'

    helpers do
      def json_params
        @json_params ||= JSON.parse(request.body.tap(&:rewind).read, symbolize_names: true)
      rescue JSON::ParserError
        halt 400, "Invalid format."
      end

      def db
        DB.get_db
      end

      def db_transaction(&block)
        db.query('BEGIN')
        done = false
        retval = block.call
        db.query('COMMIT')
        done = true
        return retval
      ensure
        db.query('ROLLBACK') unless done
      end

      def halt_error(*args)
        content_type 'text/plain'
        halt(*args)
      end

      def user_data
        [
          session.fetch(:user_id),
          session.fetch(:user_name),
          session.fetch(:is_admin),
        ]
      end
    end

    # Initialize POST /initialize 初期化エンドポイント
    post '/initialize' do
      db_for_init = DB.get_db(:batch)

      files = %w(
        1_schema.sql
        2_init.sql
        3_sample.sql
      )

      files.each do |file|
        data = File.read(File.join(SQL_DIRECTORY, file))
        db_for_init.query(data)
        db_for_init.abandon_results!
      end

      system 'rm', '-rf', ASSIGNMENTS_DIRECTORY, in: File::NULL, out: File::NULL, err: File::NULL, exception: true
      system 'cp', '-r', INIT_DATA_DIRECTORY, ASSIGNMENTS_DIRECTORY, in: File::NULL, out: File::NULL, err: File::NULL, exception: true

      content_type :json
      {language: 'ruby'}.to_json
    end

    set(:login) do |_|
      condition do
        unless session[:user_id]
          halt 401, "You are not logged in."
        end
        true
      end
    end

    set(:admin) do |_|
      condition do
        unless session[:is_admin]
          halt 403, "You are not admin user."
        end
        true
      end
    end

    # Login POST /login ログイン
    post '/login' do
      user = db.xquery('SELECT * FROM `users` WHERE `code` = ?', json_params[:code]).first
      halt 401, 'Code or Password is wrong.' unless user

      unless BCrypt::Password.new(user[:hashed_password]) == json_params[:password]
        halt 401, 'Code or Password is wrong.'
      end

      if session[:user_id] == user[:id]
        halt 400, "You are already logged in."
      end

      session[:user_id] = user[:id]
      session[:user_name] = user[:name]
      session[:is_admin] = user[:type] == TEACHER
      session.options[:path] = '/'
      session.options[:expire_after] = 3600

      ''
    end

    # Logout POST /logout ログアウト
    post '/logout' do
      session.destroy
      ''
    end

    # GetMe GET /api/users/me 自身の情報を取得
    get '/api/users/me', login: true do
      user_id, user_name, is_admin = user_data
      user_code = db.xquery('SELECT `code` FROM `users` WHERE `id` = ?', user_id).first[:code]

      content_type :json
      {
        code: user_code,
        name: user_name,
        is_admin: is_admin,
      }.to_json
    end

    # GetRegisteredCourses GET /api/users/me/courses 履修中の科目一覧取得
    get '/api/users/me/courses', login: true do
      user_id, _user_name, _is_admin = user_data

      res = db_transaction do
        courses = db.xquery(
          "SELECT `courses`.*" \
          " FROM `courses`" \
          " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" \
          " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
          STATUS_CLOSED, user_id,
        )

        courses.map do |course|
          teacher = db.xquery('SELECT * FROM `users` WHERE `id` = ?', course[:teacher_id]).first
          raise unless teacher

          {
            id: course[:id],
            name: course[:name],
            teacher: teacher[:name],
            period: course[:period],
            day_of_week: course[:day_of_week],
          }
        end
      end

      content_type :json
      res.to_json
    end

    # RegisterCourses PUT /api/users/me/courses 履修登録

    RegisterCoursesErrorResponse = Struct.new(:course_not_found, :not_registrable_status, :schedule_conflict) do
      def as_json
        h = {}
        h[:course_not_found] = course_not_found unless course_not_found.empty?
        h[:not_registrable_status] = not_registrable_status unless not_registrable_status.empty?
        h[:schedule_conflict] = schedule_conflict unless schedule_conflict.empty?
        h
      end
    end

    put '/api/users/me/courses', login: true do
      user_id, _user_name, _is_admin = user_data
      halt 400, 'Invalid format.' if !json_params.kind_of?(Array) || json_params.any? { |_| !_[:id].kind_of?(String) }

      json_params.sort! { |a,b| a.fetch(:id) <=> b.fetch(:id) }

      db_transaction do
        errors = RegisterCoursesErrorResponse.new([], [], [])
        newly_added = []
        json_params.each do |course_req|
          course_id = course_req.fetch(:id)
          course = db.xquery("SELECT * FROM `courses` WHERE `id` = ? FOR SHARE", course_id).first
          unless course
            errors.course_not_found.push(course_id)
            next
          end

          unless course[:status] == STATUS_REGISTRATION
            errors.not_registrable_status.push(course[:id])
            next
          end

          # すでに履修登録済みの科目は無視する
          count = db.xquery( "SELECT COUNT(*) AS `cnt` FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?" ,course[:id], user_id).first[:cnt]
          next if count > 0

          newly_added.push(course)
        end

        already_registered = db.xquery(
          "SELECT `courses`.*" \
          " FROM `courses`" \
          " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" \
          " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
          STATUS_CLOSED, user_id,
        ).to_a

        already_registered.concat(newly_added)
        newly_added.each do |course1|
          already_registered.each do |course2|
            if course1[:id] != course2[:id] && course1[:period] == course2[:period] && course1[:day_of_week] == course2[:day_of_week]
              errors.schedule_conflict.push(course1[:id])
              break
            end
          end
        end

        if !errors.course_not_found.empty? || !errors.not_registrable_status.empty? || !errors.schedule_conflict.empty?
          content_type :json
          halt 400, errors.as_json.to_json
        end

        newly_added.each do |course|
          db.xquery("INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `course_id` = VALUES(`course_id`), `user_id` = VALUES(`user_id`)", course[:id], user_id)
        end
      end

      ''
    end

    # GetGrades GET /api/users/me/grades 成績取得
    get '/api/users/me/grades', login: true do
      user_id, _user_name, _is_admin = user_data

      registered_courses = db.xquery(
        "SELECT `courses`.*" \
        " FROM `registrations`" \
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" \
        " WHERE `user_id` = ?",
        user_id,
      )

      # 科目毎の成績計算処理
      course_results = []
      my_gpa = 0.0
      my_credits = 0

      registered_courses.each do |course|
        # 講義一覧の取得
        classes = db.xquery(
          "SELECT *" \
          " FROM `classes`" \
          " WHERE `course_id` = ?" \
          " ORDER BY `part` DESC",
          course[:id]
        )


        # 講義毎の成績計算処理
        my_total_score = 0
        class_scores = classes.map do |klass|
          submissions_count = db.xquery( "SELECT COUNT(*) AS `cnt` FROM `submissions` WHERE `class_id` = ?", klass[:id]).first[:cnt]
          my_score = db.xquery( "SELECT `submissions`.`score` FROM `submissions` WHERE `user_id` = ? AND `class_id` = ?", user_id, klass[:id]).first&.fetch(:score)
          unless my_score
            {
              class_id: klass[:id],
              part: klass[:part],
              title: klass[:title],
              score: nil,
              submitters: submissions_count,
            }
          else
            my_total_score += my_score
            {
              class_id: klass[:id],
              part: klass[:part],
              title: klass[:title],
              score: my_score,
              submitters: submissions_count,
            }
          end
        end

        # この科目を履修している学生のTotalScore一覧を取得
        totals = db.xquery(
          "SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`" \
          " FROM `users`" \
          " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" \
          " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" \
          " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`" \
          " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`" \
          " WHERE `courses`.`id` = ?" \
          " GROUP BY `users`.`id`",
          course[:id]
        ).map { |_| _[:total_score] }

        course_results.push(
          name: course[:name],
          code: course[:code],
          total_score: my_total_score,
          total_score_t_score: Util.t_score(my_total_score, totals),
          total_score_avg: Util.average(totals, 0),
          total_score_max: Util.max(totals, 0),
          total_score_min: Util.min(totals, 0),
          class_scores: class_scores,
        )

        # 自分のGPA計算
        if course[:status] == STATUS_CLOSED
          my_gpa += my_total_score * course[:credit]
          my_credits += course[:credit]
        end
      end

      if my_credits > 0
        my_gpa = my_gpa / 100 / my_credits
      end

      # GPAの統計値
      # 一つでも修了した科目がある学生のGPA一覧
      gpas = db.xquery(
        "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit`), 0) / 100 / `credits`.`credits` AS `gpa`" \
        " FROM `users`" \
        " JOIN (" \
        "     SELECT `users`.`id` AS `user_id`, SUM(`courses`.`credit`) AS `credits`" \
        "     FROM `users`" \
        "     JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" \
        "     JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?" \
        "     GROUP BY `users`.`id`" \
        " ) AS `credits` ON `credits`.`user_id` = `users`.`id`" \
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" \
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?" \
        " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`" \
        " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`" \
        " WHERE `users`.`type` = ?" \
        " GROUP BY `users`.`id`",
        STATUS_CLOSED, STATUS_CLOSED, STUDENT,
      ).map { |_| _[:gpa].to_f }

      content_type :json
      {
        summary: {
          credits: my_credits,
          gpa: my_gpa,
          gpa_t_score: Util.t_score(my_gpa, gpas),
          gpa_avg: Util.average(gpas, 0),
          gpa_max: Util.max(gpas, 0),
          gpa_min: Util.min(gpas, 0),
        },
        courses: course_results,
      }.to_json
    end

    # SearchCourses GET /api/courses 科目検索
    get '/api/courses', login: true do
      query = "SELECT `courses`.*, `users`.`name` AS `teacher`" \
        " FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" \
        " WHERE 1=1"
      condition = ""
      args = []

      # 無効な検索条件はエラーを返さず無視して良い

      if params[:type] && !params[:type].empty?
        condition <<  " AND `courses`.`type` = ?"
        args << params[:type]
      end

      credit = Integer(params[:credit]) rescue nil
      if credit
        condition << " AND `courses`.`credit` = ?"
        args << credit
      end

      if params[:teacher] && !params[:teacher].empty?
        condition <<  " AND `users`.`name` = ?"
        args << params[:teacher]
      end

      period = Integer(params[:period]) rescue nil
      if period
        condition << " AND `courses`.`period` = ?"
        args << period
      end

      if params[:day_of_week] && !params[:day_of_week].empty?
        condition <<  " AND `courses`.`day_of_week` = ?"
        args << params[:day_of_week]
      end

      if params[:keywords] && !params[:keywords].empty?
        arr = params[:keywords].split(' ')
        name_condition = ""
        arr.each do |keyword|
          name_condition << " AND `courses`.`name` LIKE ?"
          args << "%#{keyword}%"
        end

        keywords_condition = ""
        arr.each do |keyword|
          keywords_condition += " AND `courses`.`keywords` LIKE ?"
          args << "%#{keyword}%"
        end

        condition << " AND ((1=1#{name_condition}) OR (1=1#{keywords_condition}))"
      end

      if params[:status] && !params[:status].empty?
        condition << " AND `courses`.`status` = ?"
        args << params[:status]
      end

      condition += " ORDER BY `courses`.`code`"

      page = unless params[:page]
        1
      else
        Integer(params[:page]) rescue halt 400, "Invalid page."
      end
      limit = 20
      offset = limit * (page - 1)

      # limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
      condition += " LIMIT #{(limit+1).to_i} OFFSET #{offset.to_i}"

      res = db.xquery(query+condition, *args).to_a
      links = []

      link_url = URI.parse(request.url).yield_self { |u| URI::Generic.build(path: u.path, query: u.query) }
      q = link_url.query ? URI.decode_www_form(link_url.query) : []

      if page > 1
        q.assoc('page')[1] = (page-1).to_s
        link_url.query = URI.encode_www_form(q)
        links.push(%|<#{link_url}>; rel="prev"|)
      end
      if res.size > limit
        page_param = q.assoc('page')
        unless page_param
          page_param = ['page']
          q.push(page_param)
        end
        page_param[1] = (page+1).to_s
        link_url.query = URI.encode_www_form(q)
        links.push(%|<#{link_url}>; rel="next"|)
      end

      unless links.empty?
        response.headers['Link'] = links.join(', ')
      end

      if res.size == (limit+1)
        res = res[0...(res.size-1)]
      end

      content_type :json
      res.map(&:to_h).to_json
    end

    class CourseConflict < StandardError; end

    # AddCourse POST /api/courses 新規科目登録
    post '/api/courses', login: true, admin: true do
      user_id, _user_name, _is_admin = user_data

      if !json_params.kind_of?(Hash) \
          || !json_params[:code].kind_of?(String) \
          || !json_params[:type].kind_of?(String) \
          || !json_params[:name].kind_of?(String) \
          || !json_params[:description].kind_of?(String) \
          || !json_params[:credit].kind_of?(Integer) \
          || !json_params[:period].kind_of?(Integer) \
          || !json_params[:day_of_week].kind_of?(String) \
          || !json_params[:keywords].kind_of?(String)
        halt 400, "Invalid format."
      end
      halt 400, "Invalid course type." unless [LIBERAL_ARTS, MAJOR_SUBJECTS].include?(json_params[:type])
      halt 400, "Invalid day of week." unless [MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY].include?(json_params[:day_of_week]) 

      begin
        course_id = ULID.generate
        begin
          db.xquery(
            "INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            course_id, *json_params.values_at(:code, :type, :name, :description, :credit, :period, :day_of_week), user_id, json_params[:keywords]
          )
        rescue Mysql2::Error => e
          raise CourseConflict if e.error_number == MYSQL_ERR_NUM_DUPLICATE_ENTRY
          raise
        end

        status 201
        content_type :json
        {id: course_id}.to_json
      rescue CourseConflict
        course = db.xquery( "SELECT * FROM `courses` WHERE `code` = ?", json_params[:code]).first
        raise unless course
        if %i(type name description credit period day_of_week keywords).any? { |k| json_params[k] != course[k] }
          halt 409, "A course with the same code already exists."
        end

        status 201
        content_type :json
        {id: course[:id]}.to_json
      end
    end

    # GetCourseDetail GET /api/courses/:courseID 科目詳細の取得
    get '/api/courses/:course_id', login: true do
      res = db.xquery(
        "SELECT `courses`.*, `users`.`name` AS `teacher`" \
        " FROM `courses`" \
        " JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" \
        " WHERE `courses`.`id` = ?",
        params[:course_id]
      ).first

      halt 404, "No such course." unless res

      content_type :json
      res.to_h.to_json
    end

    # SetCourseStatus PUT /api/courses/:courseID/status 科目のステータスを変更
    put '/api/courses/:course_id/status', login: true, admin: true do
      halt 400, "Invalid format" unless json_params[:status].kind_of?(String)

      db_transaction do
        count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ?", params[:course_id]).first[:cnt]
        halt 404, "No such course." if count == 0

        db.xquery("UPDATE `courses` SET `status` = ? WHERE `id` = ?", json_params[:status], params[:course_id])
      end

      ''
    end

    # GetClasses GET /api/courses/:courseID/classes 科目に紐づく講義一覧の取得
    get '/api/courses/:course_id/classes', login: true do
      user_id, _user_name, _is_admin = user_data

      classes = db_transaction do 
        count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ?", params[:course_id]).first[:cnt]
        halt 404, "No such course." if count == 0

        db.xquery(
          "SELECT `classes`.*, `submissions`.`user_id` IS NOT NULL AS `submitted`" \
          " FROM `classes`" \
            " LEFT JOIN `submissions` ON `classes`.`id` = `submissions`.`class_id` AND `submissions`.`user_id` = ?" \
            " WHERE `classes`.`course_id` = ?" \
            " ORDER BY `classes`.`part`",
            user_id, params[:course_id]
        ).to_a
      end

      # 結果が0件の時は空配列を返却
      content_type :json
      classes.map do |klass|
        {
          id: klass[:id],
          part: klass[:part],
          title: klass[:title],
          description: klass[:description],
          submission_closed: klass[:submission_closed],
          submitted: klass[:submitted] == 1, # cast_booleans doesn't work as it is a computed value  
        }
      end.to_json
    end

    # AddClass POST /api/courses/:courseID/classes 新規講義(&課題)追加
    class ClassConflict < StandardError; end
    post '/api/courses/:course_id/classes', login: true, admin: true do
      if !json_params[:part].kind_of?(Integer) || !json_params[:title].kind_of?(String) || !json_params[:description].kind_of?(String)
        halt 400, "Invalid format."
      end

      begin
        class_id = nil
        db_transaction do
          course = db.xquery("SELECT * FROM `courses` WHERE `id` = ? FOR SHARE", params[:course_id]).first
          halt 404, "No such course." unless course
          halt 400, "This course is not in-progress." if course[:status] != STATUS_IN_PROGRESS

          class_id = ULID.generate
          begin
            db.xquery(
              "INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`) VALUES (?, ?, ?, ?, ?)",
              class_id,
              params[:course_id],
              *json_params.values_at(:part, :title, :description),
            )
          rescue Mysql2::Error => e
            raise ClassConflict if e.error_number == MYSQL_ERR_NUM_DUPLICATE_ENTRY
            raise
          end
        end
        status 201
        content_type :json
        {class_id: class_id}.to_json
      rescue ClassConflict
        klass = db.xquery("SELECT * FROM `classes` WHERE `course_id` = ? AND `part` = ?", params[:course_id], json_params[:part]).first
        raise unless klass
        if json_params[:title] != klass[:title] || json_params[:description] != klass[:description]
          halt 409, "A class with the same part already exists."
        end
        status 201
        content_type :json
        {class_id: klass[:id]}.to_json
      end
    end

    # SubmitAssignment POST /api/courses/:courseID/classes/:classID/assignments 課題の提出
    post '/api/courses/:course_id/classes/:class_id/assignments', login: true do
      user_id, _user_name, _is_admin = user_data

      course_id, class_id = params[:course_id], params[:class_id]

      db_transaction do
        status = db.xquery( "SELECT `status` FROM `courses` WHERE `id` = ? FOR SHARE", course_id).first&.fetch(:status)
        halt 404, "No such course." unless status

        halt 400, "This course is not in progress." unless status == STATUS_IN_PROGRESS

        registration_count = db.xquery(
          "SELECT COUNT(*) AS `cnt` FROM `registrations` WHERE `user_id` = ? AND `course_id` = ?",
          user_id, course_id,
        ).first[:cnt]
        halt 400, "You have not taken this course." if registration_count == 0


        submission_closed = db.xquery("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE", class_id).first&.fetch(:submission_closed)
        halt 404, "No such class." if submission_closed == nil
        halt 400, "Submission has been closed for this class." if submission_closed

        file = params[:file]
        halt 400, 'Invalid file.' if file && (!file.kind_of?(Hash) || !file[:tempfile].is_a?(Tempfile))

        db.xquery("INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `file_name` = VALUES(`file_name`)", user_id, class_id, file.fetch(:filename))

        data = file.fetch(:tempfile).binmode.read
        dst = File.join(ASSIGNMENTS_DIRECTORY, "#{class_id}-#{user_id}.pdf")
        File.binwrite dst, data
      end

      status 204
      ''
    end

    # RegisterScores PUT /api/courses/:courseID/classes/:classID/assignments/scores 採点結果登録
    put '/api/courses/:course_id/classes/:class_id/assignments/scores', login: true, admin: true do
      class_id = params[:class_id]

      db_transaction do
        submission_closed = db.xquery("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE", class_id).first&.fetch(:submission_closed)
        halt 404, "No such class." if submission_closed == nil
        halt 400, "This assignment is not closed yet." unless submission_closed

        if !json_params.kind_of?(Array) || json_params.any? { |score| !score[:user_code].kind_of?(String) || !score[:score].kind_of?(Integer) }
          halt 400, "Invalid format."
        end

        json_params.each do |score|
          db.xquery("UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?", score[:score], score[:user_code], class_id)
        end
      end

      status 204
      ''
    end

    # DownloadSubmittedAssignments GET /api/courses/:courseID/classes/:classID/assignments/export 提出済みの課題ファイルをzip形式で一括ダウンロード
    get '/api/courses/:course_id/classes/:class_id/assignments/export', login: true, admin: true do
      class_id = params[:class_id]

      zip_file_path = nil
      db_transaction do
        class_count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `classes` WHERE `id` = ? FOR UPDATE", class_id).first[:cnt]
        halt 404, "No such class." if class_count == 0

        submissions = db.xquery(
          "SELECT `submissions`.`user_id`, `submissions`.`file_name`, `users`.`code` AS `user_code`" \
          " FROM `submissions`" \
          " JOIN `users` ON `users`.`id` = `submissions`.`user_id`" \
          " WHERE `class_id` = ?",
          class_id,
        )

        zip_file_path = File.join(ASSIGNMENTS_DIRECTORY, "#{class_id}.zip")
        create_submissions_zip(zip_file_path, class_id, submissions)

        db.xquery("UPDATE `classes` SET `submission_closed` = true WHERE `id` = ?", class_id)
      end

      content_type 'application/zip'
      send_file zip_file_path
    end

    def create_submissions_zip(zip_file_path, class_id, submissions)
      tmp_dir = File.join(ASSIGNMENTS_DIRECTORY, class_id, '')
      system "rm", "-rf", tmp_dir, in: File::NULL, out: File::NULL, err: File::NULL,exception: true
      system "mkdir", tmp_dir, in: File::NULL, out: File::NULL, err: File::NULL, exception: true

      # ファイル名を指定の形式に変更
      submissions.each do |submission|
        system(
          "cp",
          File.join(ASSIGNMENTS_DIRECTORY, "#{class_id}-#{submission[:user_id]}.pdf"),
          File.join(tmp_dir, "#{submission[:user_code]}-#{submission[:file_name]}"),
          in: File::NULL,
          out: File::NULL,
          err: File::NULL,
          exception: true,
        )
      end

      # -i 'tmpDir/*': 空zipを許す
      system "zip", "-j", "-r", zip_file_path, tmp_dir, "-i", "#{tmp_dir}*", in: File::NULL, out: File::NULL, err: File::NULL, exception: true
    end


    # GetAnnouncementList GET /api/announcements お知らせ一覧取得
    get '/api/announcements', login: true do
      user_id, _user_name, _is_admin = user_data

      page, limit, announcements, unread_count = db_transaction do
        query = "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, NOT `unread_announcements`.`is_deleted` AS `unread`" \
          " FROM `announcements`" \
          " JOIN `courses` ON `announcements`.`course_id` = `courses`.`id`" \
          " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" \
          " JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id`" \
          " WHERE 1=1"
        args = []

        if params[:course_id] && !params[:course_id].empty?
          query.concat(" AND `announcements`.`course_id` = ?")
          args.push(params[:course_id])
        end

        query.concat(
          " AND `unread_announcements`.`user_id` = ?" \
          " AND `registrations`.`user_id` = ?" \
          " ORDER BY `announcements`.`id` DESC"
        )
        args.push(user_id, user_id)

        page = unless params[:page]
          1
        else
          Integer(params[:page]) rescue halt 400, "Invalid page."
        end
        limit = 20
        offset = limit * (page - 1)

        # limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
        query.concat " LIMIT #{(limit+1).to_i} OFFSET #{offset.to_i}"

        announcements = db.xquery(query, *args).map do |a|
          {
            id: a[:id],
            course_id: a[:course_id],
            course_name: a[:course_name],
            title: a[:title],
            unread: a[:unread] == 1, # cast_booleans doesn't work as it is a computed value
          }
        end
        unread_count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `unread_announcements` WHERE `user_id` = ? AND NOT `is_deleted`", user_id).first[:cnt]

        [page, limit, announcements, unread_count]
      end

      links = []

      link_url = URI.parse(request.url).yield_self { |u| URI::Generic.build(path: u.path, query: u.query) }
      q = link_url.query ? URI.decode_www_form(link_url.query) : []

      if page > 1
        q.assoc('page')[1] = (page-1).to_s
        link_url.query = URI.encode_www_form(q)
        links.push(%|<#{link_url}>; rel="prev"|)
      end
      if announcements.size > limit
        page_param = q.assoc('page')
        unless page_param
          page_param = ['page']
          q.push(page_param)
        end
        page_param[1] = (page+1).to_s
        link_url.query = URI.encode_www_form(q)
        links.push(%|<#{link_url}>; rel="next"|)
      end

      unless links.empty?
        response.headers['Link'] = links.join(', ')
      end

      if announcements.size == (limit+1)
        announcements = announcements[0...(announcements.size-1)]
      end

      # 対象になっているお知らせが0件の時は空配列を返却
      content_type :json
      {
        unread_count: unread_count,
        announcements: announcements,
      }.to_json
    end

    class AnnouncementConflict < StandardError; end

    # AddAnnouncement POST /api/announcements 新規お知らせ追加
    post '/api/announcements', login: true, admin: true do
      if !json_params.kind_of?(Hash) || %i(id course_id title message).any? { |k| !json_params[k].kind_of?(String) }
        halt 400, "Invalid format."
      end

      begin
        db_transaction do
          count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ?", json_params[:course_id]).first[:cnt]
          halt 404, "No such course." if count == 0

          begin
            db.xquery(
              "INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`) VALUES (?, ?, ?, ?)",
              *json_params.values_at(:id, :course_id, :title, :message),
            )
          rescue Mysql2::Error => e
            raise AnnouncementConflict if e.error_number == MYSQL_ERR_NUM_DUPLICATE_ENTRY
            raise
          end

          targets = db.xquery(
            "SELECT `users`.* FROM `users`" \
            " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" \
            " WHERE `registrations`.`course_id` = ?",
            json_params[:course_id],
          )
          targets.each do |user|
            db.xquery("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`) VALUES (?, ?)", json_params[:id], user[:id])
          end
        end
      rescue AnnouncementConflict
        announcement = db.xquery("SELECT * FROM `announcements` WHERE `id` = ?", json_params[:id])
        raise unless announcement
        if %i(course_id title message).any? { |k| announcement[k] != json_params[k] }
          halt 409, "An announcement with the same id already exists."
        end
        status 201
        ''
      end

      status 201
      ''
    end

    # GetAnnouncementDetail GET /api/announcements/:announcementID お知らせ詳細取得
    get '/api/announcements/:announcement_id', login: true do
      user_id, _user_name, _is_admin = user_data

      announcement_id = params[:announcement_id]

      announcement = db_transaction do
        announcement = db.xquery(
          "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `announcements`.`message`, NOT `unread_announcements`.`is_deleted` AS `unread`" \
          " FROM `announcements`" \
          " JOIN `courses` ON `courses`.`id` = `announcements`.`course_id`" \
          " JOIN `unread_announcements` ON `unread_announcements`.`announcement_id` = `announcements`.`id`" \
          " WHERE `announcements`.`id` = ?" \
          " AND `unread_announcements`.`user_id` = ?",
          announcement_id, user_id,
        ).first
        halt 404, "No such announcement." unless announcement

        registration_count = db.xquery("SELECT COUNT(*) AS `cnt` FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", announcement[:course_id], user_id).first[:cnt]
        halt 404, "No such announcement." if registration_count == 0

        db.xquery("UPDATE `unread_announcements` SET `is_deleted` = true WHERE `announcement_id` = ? AND `user_id` = ?", announcement_id, user_id)

        {
          id: announcement[:id],
          course_id: announcement[:course_id],
          course_name: announcement[:course_name],
          title: announcement[:title],
          message: announcement[:message],
          unread: announcement[:unread] == 1, # cast_booleans doesn't work as it is a computed value
        }
      end

      content_type :json
      announcement.to_json
    end

  end
end
