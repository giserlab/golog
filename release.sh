cd bin
for file in *; do
# 只处理普通文件（跳过目录和已有的压缩包）
if [ -f "$file" ] && [[ "$file" != *.tar.gz ]] && [[ "$file" != *.deb ]]; then
    tar czf "${file}.tar.gz" "$file"
fi
done