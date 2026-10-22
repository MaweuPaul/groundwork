import ee

def add_ndvi(image: ee.Image) -> ee.Image:
    """Computes NDVI and adds it as a band."""
    ndvi = image.normalizedDifference(['B8', 'B4']).rename('ndvi')
    return image.addBands(ndvi)
