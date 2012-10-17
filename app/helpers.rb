Wallpapers.helpers do
  def storage_connection
    return Storage.connection
  end

  def get_files
    return storage_connection.files
  end

  def get_file filename
    return get_files.get(filename)
  end
end
