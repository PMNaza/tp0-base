FROM python:3.9.7-slim
COPY server / 
# La copia del archivo de configuracion no se copiará a la imagen ya que será ignorado por el dockerignore (Que solo afecta en tiempo de construcción de la imagen)
RUN python -m unittest tests/test_common.py
ENTRYPOINT ["/bin/sh"]
