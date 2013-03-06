Wallpapers.controllers  do
  PERPAGE = 30

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

    if @image.nil?
      404
    else
      if @image.public_url
        redirect @image.public_url
      else
        logger.warn "Image does not have a public url: #{@image.inspect}"
        403
      end
    end
  end

  get '/thumbnail/:id', :cache => true do
    expires_in 86400 # 1 day

    begin
      raise if !params["reset"].nil?

      image = Storage.get_thumb params[:id]

      logger.push "Redirect #{params[:id]} to #{image.public_url.inspect}", :info
      redirect image.public_url
    rescue
      @image = Storage.get_file params[:id]
      thumbnail = MiniMagick::Image.read(@image.body)
      thumbnail.combine_options do |c|
        c.quality "60"
        c.resize "300x200"
      end

      thumbnail_file = File.join(File.dirname(__FILE__), "../tmp", "thumb_#{params[:id]}")

      thumbnail.write thumbnail_file

      file = Storage.thumb_dir.files.create(
        :key    => params[:id],
        :body   => File.open(thumbnail_file),
        :public => true,
      )

      redirect file.public_url
    end
  end

  get '/403' do
    403
  end

  get '/404' do
    404
  end

  error 400..510 do
    @code = response.status
    render :error
  end
end
