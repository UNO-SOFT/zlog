# logr -> zlog
    sed -i -e '/logr/ { s,go-logr/logr,UNO-SOFT/zlog/v2,; s/logr.Logger/*slog.Logger/; s/logr.FromContext\(OrDiscard\)*/zlog.SFromContext/; s/logr.NewContext/zlog.NewSContext/; }; /logger\.Error([^"]/ s/logger\.Error(\([^,]*\),\(.*\))$/logger.Error(\2, "error", \1)/; /logger\.V/ s/logger\.V([0-9][0-9]*)\.Info/logger.Debug/; /logger\.With/ { s/WithName/WithGroup/; s/WithValues/With/; }' $(fgrep -l logr $(find . -type f -name '*.go'))