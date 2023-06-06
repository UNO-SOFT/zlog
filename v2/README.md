# logr -> zlog
    sed -i -e '/logr/ { s,go-logr/logr,UNO-SOFT/zlog/v2,; s/logr.FromContextOrDiscard/zlog.SFromContext/; s/logr.NewContext/zlog.NewSContext/; }; /logger\.Error/ s/logger\.Error(\([^,]*\),\(.*\))$/logger.Error(\2, "error", \1)/; /logger\.V/ s/logger\.V([0-9][0-9]*)\.Info/logger.Debug/' $(fgrep -l logr $(find . -type f -name '*.go'))
