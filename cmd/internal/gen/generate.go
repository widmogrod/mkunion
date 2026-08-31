package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/generators"
	"github.com/widmogrod/mkunion/x/shape"
)

// Generate runs the full mkunion pipeline (unions, serde, shapes, match, and
// optionally the type registry) for the given Go source files. It returns the
// paths of all files it wrote.
func Generate(sourcePaths []string, typeRegistry bool) ([]string, error) {
	packages := make(map[string]*shape.InferredInfo)
	var savedFiles []string

	// shapes already generated in this run, keyed by fully qualified type name,
	// so a shape referenced from many files is generated only once per run
	generatedShapes := map[string]bool{}

	for _, sourcePath := range sourcePaths {
		inferred, err := shape.InferFromFile(sourcePath)
		if err != nil {
			return savedFiles, err
		}

		if _, ok := packages[inferred.PackageImportName()]; !ok {
			packages[inferred.PackageImportName()] = inferred
		}

		outputs := []struct {
			infix    string
			generate func(*shape.InferredInfo) (bytes.Buffer, error)
		}{
			{"union_gen", generateUnions},
			{"serde_gen", generateSerde},
			{"shape_gen", func(inferred *shape.InferredInfo) (bytes.Buffer, error) {
				return generateShape(inferred, generatedShapes)
			}},
			{"match_gen", generateMatch},
		}

		for _, output := range outputs {
			contents, err := output.generate(inferred)
			if err != nil {
				return savedFiles, fmt.Errorf("failed generating %s in %s: %w", output.infix, sourcePath, err)
			}
			savedFile, err := SaveFile(contents, sourcePath, output.infix)
			if err != nil {
				return savedFiles, fmt.Errorf("failed saving %s in %s: %w", output.infix, sourcePath, err)
			}
			if len(savedFile) > 0 {
				savedFiles = append(savedFiles, savedFile)
			}
		}
	}

	if typeRegistry {
		uniqueDirs := make(map[string]bool)
		for _, inferred := range packages {
			dir := path.Dir(inferred.FileName())
			uniqueDirs[dir] = true
		}

		var dirs []string
		for dir := range uniqueDirs {
			dirs = append(dirs, dir)
		}

		savedFiles2, err := GenerateTypeRegistryForDir(dirs)
		if err != nil {
			return savedFiles, err
		}
		savedFiles = append(savedFiles, savedFiles2...)
	}

	return savedFiles, nil
}

// GenerateTypeRegistryForDir generates the types_reg_gen.go type registry for
// each directory that has indexed shapes and does not opt out with the
// no-type-registry tag option.
func GenerateTypeRegistryForDir(uniqueDirs []string) ([]string, error) {
	var savedFiles []string
	for _, dir := range uniqueDirs {
		indexed, err := shape.NewIndexTypeInDir(dir)
		if err != nil {
			return savedFiles, fmt.Errorf("gen: failed indexing types in directory %s: %w", dir, err)
		}

		if shape.TagHasOption(indexed.PackageTags(), shape.TagUnionName, shape.TagUnionOptionNoRegistry) {
			continue
		}

		if len(indexed.IndexedShapes()) < 1 {
			continue
		}

		contents, err := generators.GenerateTypeRegistry(indexed, shape.LookupShapeOnDisk)
		if err != nil {
			return savedFiles, fmt.Errorf("gen: failed walking through directory %s: %w", dir, err)
		}

		regPath := path.Join(dir, "types.go")
		savedFile, err := SaveFile(contents, regPath, "reg_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("gen: failed saving type registry in %s: %w", regPath, err)
		}

		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}
	}

	return savedFiles, nil
}

func generateUnions(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	unions := inferred.RetrieveUnions()
	if len(unions) == 0 {
		return shapesContents, nil
	}

	var err error
	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0)

	for _, union := range unions {
		packageName = shape.ToGoPkgName(union)

		genVisitor := generators.NewVisitorGenerator(union)
		genVisitor.SkipImportsAndPackage(true)

		contents, err := genVisitor.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("failed to generate visitor for %s: %w", shape.ToGoTypeName(union), err)
		}
		shapesContents.Write(contents)

		if shape.TagHasOption(union.Tags, "mkunion", "noserde") {
			continue
		}

		genSerde := generators.NewSerdeJSONUnion(union)
		genSerde.SkipImportsAndPackage(true)

		contents, err = genSerde.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("gen.generateUnions: failed to generate json serde for %s: %w", shape.ToGoTypeName(union), err)
		}
		shapesContents.Write(contents)

		pkgMap = generators.MergePkgMaps(pkgMap,
			genSerde.ExtractImports(union),
		)
	}

	contents := bytes.Buffer{}
	contents.WriteString(generators.Header + "\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))
	_, err = shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("gen.generateUnions: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func generateSerde(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	shapes := inferred.RetrieveShapesTaggedAs("serde")
	if len(shapes) == 0 {
		return shapesContents, nil
	}

	var err error
	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0)

	for _, x := range shapes {
		packageName = shape.ToGoPkgName(x)
		genSerde := generators.NewSerdeJSONTagged(x)
		genSerde.SkipImportsAndPackage(true)

		contents, err := genSerde.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("gen.generateSerde: failed to generate json serde for %s: %w", shape.ToGoTypeName(x), err)
		}
		shapesContents.WriteString(contents)

		pkgMap = generators.MergePkgMaps(pkgMap,
			genSerde.ExtractImports(x),
		)
	}

	contents := bytes.Buffer{}
	contents.WriteString(generators.Header + "\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))

	_, err = shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("gen.generateSerde: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func generateShape(inferred *shape.InferredInfo, generatedShapes map[string]bool) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	shapes := inferred.RetrieveShapes()
	if len(shapes) == 0 {
		return shapesContents, nil
	}

	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0)

	for _, x := range shapes {
		if shape.TagGetValue(shape.Tags(x), shape.TagShapeName, "") == "-" {
			// skip shape generation for this type
			continue
		}

		packageName = shape.ToGoPkgName(x)
		contents, err := generateShapeFollow(x, &pkgMap, &initFunc, inferred, generatedShapes)
		if err != nil {
			return shapesContents, fmt.Errorf("gen.generateShape: failed to generate shape for %s: %w", shape.ToGoTypeName(x), err)
		}
		if contents != nil {
			_, err = contents.WriteTo(&shapesContents)
			if err != nil {
				return shapesContents, fmt.Errorf("gen.generateShape: failed to write shape for %s: %w", shape.ToGoTypeName(x), err)
			}
		}
	}

	if len(shapesContents.Bytes()) == 0 {
		return shapesContents, nil
	}

	contents := bytes.Buffer{}
	contents.WriteString(generators.Header + "\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))
	_, err := shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("gen.generateShape: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func generateShapeFollow(x shape.Shape, pkgMap *generators.PkgMap, initFunc *[]string, inferred *shape.InferredInfo, generatedShapes map[string]bool) (*bytes.Buffer, error) {
	var result *bytes.Buffer
	for _, y := range shape.ExtractRefs(x) {
		// filter types that are not from the same package
		if y.PkgImportName != shape.ToGoPkgImportName(x) {
			log.Debugf("gen.generateShapeFollow: skipping %s, not from the same package", shape.ToGoTypeName(y))
			continue
		}

		contents, err := generateShapeOnce(y, pkgMap, initFunc, inferred, generatedShapes)
		if err != nil {
			return nil, fmt.Errorf("gen.generateShapeFollow: failed to generate shape for %s: %w", shape.ToGoTypeName(y), err)
		}

		if contents == nil {
			continue
		}

		if result == nil {
			result = contents
		} else {
			_, err = contents.WriteTo(result)
			if err != nil {
				return nil, fmt.Errorf("gen.generateShapeFollow: failed to write shape for %s: %w", shape.ToGoTypeName(y), err)
			}
		}
	}

	return result, nil
}

func generateShapeOnce(x shape.Shape, pkgMap *generators.PkgMap, initFunc *[]string, inferred *shape.InferredInfo, generatedShapes map[string]bool) (*bytes.Buffer, error) {
	key := shape.ToGoTypeName(x, shape.WithPkgImportName())
	if generatedShapes[key] {
		log.Debugf("gen.generateShapeOnce: shape %s already generated", key)
		return nil, nil
	}

	result := bytes.Buffer{}

	switch x := x.(type) {
	case *shape.RefName:
		y := inferred.RetrieveShapeFromRef(x)
		if y == nil {
			log.Warnf("gen.generateShapeOnce: failed to lookup shape %s", shape.ToGoTypeName(x, shape.WithPkgImportName()))
			return nil, nil
		}

		switch y := y.(type) {
		case *shape.RefName:
			log.Warnf("gen.generateShapeOnce: lookup RefName %s", shape.ToGoTypeName(y, shape.WithPkgImportName()))
			return nil, nil
		}

		return generateShapeOnce(y, pkgMap, initFunc, inferred, generatedShapes)

	case *shape.UnionLike:
		for _, v := range x.Variant {
			key := shape.ToGoTypeName(v, shape.WithPkgImportName())
			generatedShapes[key] = true
		}
	}

	generatedShapes[key] = true

	gen := generators.NewShapeTagged(x)
	gen.SkipImportsAndPackage(true)
	gen.SkipInitFunc(true)

	result.WriteString("//shape:shape\n")
	contents, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("gen.generateShapeOnce: failed to generate tagged shape for %s: %w", shape.ToGoTypeName(x, shape.WithPkgImportName()), err)
	}
	result.WriteString(contents)

	*pkgMap = generators.MergePkgMaps(*pkgMap,
		gen.ExtractImports(x),
	)

	*initFunc = append(*initFunc, gen.ExtractImportFuncs(x)...)

	return &result, nil
}

func generateMatch(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	result := bytes.Buffer{}

	match := generators.NewMkMatchTaggedNodeVisitor()
	match.FromInferredInfo(inferred)

	specs := match.Specs()
	if len(specs) == 0 {
		return result, nil
	}

	derived := generators.MkMatchGenerator{
		Header:      generators.Header,
		PackageName: inferred.PackageName(),
		MatchSpecs:  specs,
	}

	b, err := derived.Generate()
	if err != nil {
		return result, fmt.Errorf("gen.generateMatch: failed to generate match: %w", err)
	}

	result.Write(b)
	return result, nil
}

// SaveFile writes the generated contents next to sourcePath as
// <source>_<infix>.go, gofmt-formatted. Empty contents write nothing and
// return an empty file name.
func SaveFile(contents bytes.Buffer, sourcePath string, infix string) (string, error) {
	if len(contents.Bytes()) == 0 {
		return "", nil
	}

	sourceName := path.Base(sourcePath)
	baseName := strings.TrimSuffix(sourceName, path.Ext(sourceName))
	fileName := path.Join(
		path.Dir(sourcePath),
		fmt.Sprintf("%s_%s.go", baseName, infix),
	)

	// Format the generated Go code
	formatted, err := format.Source(contents.Bytes())
	if err != nil {
		// Log warning but continue with unformatted code
		log.Warnf("failed to format generated code for %s: %v", fileName, err)
		formatted = contents.Bytes()
	}

	log.Infof("writing %s", fileName)
	err = os.WriteFile(fileName, formatted, 0644)
	if err != nil {
		return fileName, fmt.Errorf("gen.SaveFile: failed to write file %s: %w", sourcePath, err)
	}
	return fileName, nil
}
