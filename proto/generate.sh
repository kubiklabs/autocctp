cd proto
buf generate
cd ..

cp -r github.com/noble-assets/autocctp/* ./
rm -rf github.com
