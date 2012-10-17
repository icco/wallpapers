Wallpapers.controllers  do
  get :index do
    @images = get_files
    erb :index, :locals => {}
  end

  get '/image/:id' do
    @image = get_file params[:id]

    # TODO: set content-type
    return @image.body
  end


  get '/thumbnail/:id' do
    @image = get_file params[:id]

    # TODO: set content-type
    # TODO: create and cache thumbnail
    return @image.body
  end
end
