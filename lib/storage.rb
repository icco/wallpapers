class Storage
  def self.connection(force_prod = false)
    credentials = if ENV["RACK_ENV"].eql?("production") || force_prod
                    {
                      provider: "Google",
                      google_project: "icco-natwelch",
                      google_storage_access_key_id: ENV["GOOGLE_KEY"],
                      google_storage_secret_access_key: ENV["GOOGLE_SECRET"],
                    }
                  else
                    {
                      provider: "Local",
                      local_root: "/tmp/",
                      endpoint: "file:///tmp/",
                    }
                  end

    Fog::Storage.new(credentials)
  end

  def self.directory(directory_name, force_prod = false)
    directory = connection(force_prod).directories.get(directory_name)

    if directory.nil?
      directory = connection.directories.create(
        key: directory_name,
        public: true
      )
    end

    directory
  end

  def self.main_dir(force_prod = false)
    directory "iccowalls", force_prod
  end

  def self.get_files(force_prod = false)
    main_dir(force_prod).files.shuffle
  end

  def self.get_range(range, force_prod = false)
    get_files(force_prod).to_a[range]
  end

  def self.get_file(filename, force_prod = false)
    get_files(force_prod).get(filename)
  end
end

module Fog
  module Storage
    class GoogleXML
      class File < Fog::Model
        def file_url
          requires :directory, :key
          "https://#{directory.key}.storage.googleapis.com/#{key}"
        end

        def cdn_url
          requires :directory, :key
          "https://icco-walls.imgix.net/#{key}?auto=compress&w=2560&h=1440&crop=entropy&fm=png"
        end

        def thumb_url
          requires :key
          "https://icco-walls.imgix.net/#{key}?w=600&h=400&fit=crop&auto=compress&fm=png"
        end
      end
    end
  end
end

module Fog
  module Storage
    class Local
      class File < Fog::Model
        def file_url
          requires :directory, :key
          "https://#{directory.key}.storage.googleapis.com/#{key}"
        end

        def cdn_url
          requires :directory, :key
          "https://icco-walls.imgix.net/#{key}?auto=compress&w=2560&h=1440&crop=entropy&fm=png"
        end

        def thumb_url
          requires :directory, :key
          "https://icco-walls.imgix.net/#{key}?w=600&h=400&fit=crop&auto=compress&fm=png"
        end
      end
    end
  end
end
