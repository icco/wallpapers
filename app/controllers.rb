Wallpapers.controllers  do
  PERPAGE = 12

  get :index do
    @images = Storage.get_range(0...PERPAGE)
    next_page = Storage.get_files.count > PERPAGE ? 2 : false
    erb :index, :locals => { :prior_page => false, :next_page => next_page }
  end

  get '/page/:id' do
    page = params[:id].to_i - 1

    if page == 0
      redirect '/'
    end

    next_page = page + 2
    prior_page = page

    @images = Storage.get_range((page*PERPAGE)...(next_page*PERPAGE))
    erb :index, :locals => { :prior_page => prior_page, :next_page => next_page }
  end

  get '/image/:id' do
    @image = Storage.get_file params[:id]
    logger.warn @image.inspect
    @image.public_url
  end

  get '/thumbnail/:id', :cache => true do
    expires_in 86400 # 1 day

    begin
      raise if !params["reset"].nil?

      image = Storage.get_thumb params[:id]

      if image.nil?
        stream = File.open "tmp/thumb_#{params[:id]}"
      end

      redirect "http://placehold.it/300x200" if Padrino.env == :development
      redirect image.public_url
    rescue
      @image = Storage.get_file params[:id]
      thumbnail = MiniMagick::Image.read(@image.body)
      thumbnail.combine_options do |c|
          c.quality "60"
          c.resize "300x200"
      end
      thumbnail.write "tmp/thumb_#{params[:id]}"

      file = Storage.thumb_dir.files.create(
        :key    => params[:id],
        :body   => File.open("tmp/thumb_#{params[:id]}"),
        :public => true,
      )

      redirect file.public_url
    end
  end
end
