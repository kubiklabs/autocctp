cd proto
buf generate --template buf.gen.gogo.yaml
buf generate --template buf.gen.pulsar.yaml
cd ..

cp -r github.com/noble-assets/autocctp/v1/* ./
rm -rf github.com
rm -rf noble
