package blastradius

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// LineageNode represents a node in the image lineage graph.
type LineageNode struct {
	ImageID      uuid.UUID
	ImageFamily  string
	ImageVersion string
	Depth        int
	Parent       *LineageNode
	Children     []*LineageNode
}

// LineageGraph represents the complete lineage graph for analysis.
type LineageGraph struct {
	Nodes map[uuid.UUID]*LineageNode
	Roots []*LineageNode
}

// BuildLineageGraph constructs a lineage graph starting from given image IDs.
func (e *Engine) BuildLineageGraph(ctx context.Context, orgID uuid.UUID, startImageIDs []uuid.UUID) (*LineageGraph, error) {
	graph := &LineageGraph{
		Nodes: make(map[uuid.UUID]*LineageNode),
	}

	// Fetch all lineage relationships for the org
	query := `
		SELECT
			il.image_id,
			il.parent_image_id,
			i.family,
			i.version
		FROM image_lineage il
		JOIN images i ON i.id = il.image_id
		WHERE i.org_id = $1
	`

	rows, err := e.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("query lineage: %w", err)
	}
	defer rows.Close()

	// Build parent-child relationships
	parentMap := make(map[uuid.UUID]uuid.UUID)     // child -> parent
	childrenMap := make(map[uuid.UUID][]uuid.UUID) // parent -> children
	type imageInfo struct {
		ImageID       uuid.UUID
		ParentImageID uuid.UUID
		Family        string
		Version       string
	}
	imageInfoMap := make(map[uuid.UUID]imageInfo)

	for rows.Next() {
		var info imageInfo
		if err := rows.Scan(&info.ImageID, &info.ParentImageID, &info.Family, &info.Version); err != nil {
			return nil, fmt.Errorf("scan lineage row: %w", err)
		}
		parentMap[info.ImageID] = info.ParentImageID
		childrenMap[info.ParentImageID] = append(childrenMap[info.ParentImageID], info.ImageID)
		imageInfoMap[info.ImageID] = info
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lineage rows: %w", err)
	}

	// Create nodes for starting images
	for _, id := range startImageIDs {
		node := &LineageNode{
			ImageID: id,
			Depth:   0,
		}
		if info, ok := imageInfoMap[id]; ok {
			node.ImageFamily = info.Family
			node.ImageVersion = info.Version
		}
		graph.Nodes[id] = node
	}

	// Expand children recursively
	visited := make(map[uuid.UUID]bool)
	var expandChildren func(node *LineageNode)
	expandChildren = func(node *LineageNode) {
		if visited[node.ImageID] {
			return
		}
		visited[node.ImageID] = true

		children := childrenMap[node.ImageID]
		for _, childID := range children {
			childNode := &LineageNode{
				ImageID: childID,
				Depth:   node.Depth + 1,
				Parent:  node,
			}
			if info, ok := imageInfoMap[childID]; ok {
				childNode.ImageFamily = info.Family
				childNode.ImageVersion = info.Version
			}
			node.Children = append(node.Children, childNode)
			graph.Nodes[childID] = childNode
			expandChildren(childNode)
		}
	}

	for _, node := range graph.Nodes {
		expandChildren(node)
	}

	// Identify root nodes (starting images without parents in the graph)
	for _, node := range graph.Nodes {
		if node.Parent == nil {
			graph.Roots = append(graph.Roots, node)
		}
	}

	return graph, nil
}

// GetAllDescendants returns all descendant image IDs from a set of starting images.
func (e *Engine) GetAllDescendants(ctx context.Context, orgID uuid.UUID, imageIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(imageIDs) == 0 {
		return nil, nil
	}

	// Recursive CTE to find all descendants
	query := `
		WITH RECURSIVE descendants AS (
			SELECT
				il.image_id,
				il.parent_image_id,
				1 as depth
			FROM image_lineage il
			JOIN images i ON i.id = il.image_id
			WHERE il.parent_image_id = ANY($1) AND i.org_id = $2

			UNION ALL

			SELECT
				il.image_id,
				il.parent_image_id,
				d.depth + 1
			FROM image_lineage il
			JOIN images i ON i.id = il.image_id
			JOIN descendants d ON d.image_id = il.parent_image_id
			WHERE i.org_id = $2 AND d.depth < 10
		)
		SELECT DISTINCT image_id FROM descendants
	`

	rows, err := e.db.Query(ctx, query, imageIDs, orgID)
	if err != nil {
		return nil, fmt.Errorf("query descendants: %w", err)
	}
	defer rows.Close()

	var descendantIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan descendant id: %w", err)
		}
		descendantIDs = append(descendantIDs, id)
	}

	return descendantIDs, rows.Err()
}

// GetLineagePath returns the lineage path from an image to its root ancestor.
func (e *Engine) GetLineagePath(ctx context.Context, imageID uuid.UUID) ([]models.AffectedImage, error) {
	query := `
		WITH RECURSIVE ancestors AS (
			SELECT
				i.id as image_id,
				i.family as image_family,
				i.version as image_version,
				il.parent_image_id,
				0 as depth
			FROM images i
			LEFT JOIN image_lineage il ON il.image_id = i.id
			WHERE i.id = $1

			UNION ALL

			SELECT
				i.id as image_id,
				i.family as image_family,
				i.version as image_version,
				il.parent_image_id,
				a.depth + 1
			FROM images i
			JOIN image_lineage il ON il.image_id = i.id
			JOIN ancestors a ON a.parent_image_id = i.id
			WHERE a.depth < 10
		)
		SELECT image_id, image_family, image_version, depth
		FROM ancestors
		ORDER BY depth DESC
	`

	rows, err := e.db.Query(ctx, query, imageID)
	if err != nil {
		return nil, fmt.Errorf("query lineage path: %w", err)
	}
	defer rows.Close()

	var path []models.AffectedImage
	for rows.Next() {
		var img models.AffectedImage
		var depth int
		if err := rows.Scan(&img.ImageID, &img.ImageFamily, &img.ImageVersion, &depth); err != nil {
			return nil, fmt.Errorf("scan path row: %w", err)
		}
		img.LineageDepth = depth
		path = append(path, img)
	}

	return path, rows.Err()
}

// FindCommonAncestor finds the lowest common ancestor of two images.
func (e *Engine) FindCommonAncestor(ctx context.Context, imageID1, imageID2 uuid.UUID) (*uuid.UUID, error) {
	path1, err := e.GetLineagePath(ctx, imageID1)
	if err != nil {
		return nil, fmt.Errorf("get path for image1: %w", err)
	}

	path2, err := e.GetLineagePath(ctx, imageID2)
	if err != nil {
		return nil, fmt.Errorf("get path for image2: %w", err)
	}

	// Convert path1 to a set for O(1) lookup
	path1Set := make(map[uuid.UUID]bool)
	for _, img := range path1 {
		path1Set[img.ImageID] = true
	}

	// Find first common ancestor
	for _, img := range path2 {
		if path1Set[img.ImageID] {
			id := img.ImageID
			return &id, nil
		}
	}

	return nil, nil
}

// ImageLineageInfo contains lineage metadata for an image.
type ImageLineageInfo struct {
	ImageID          uuid.UUID  `json:"imageId"`
	ParentCount      int        `json:"parentCount"`
	ChildCount       int        `json:"childCount"`
	TotalDescendants int        `json:"totalDescendants"`
	MaxDepth         int        `json:"maxDepth"`
	RootAncestor     *uuid.UUID `json:"rootAncestor,omitempty"`
}

// GetLineageInfo retrieves lineage statistics for an image.
func (e *Engine) GetLineageInfo(ctx context.Context, imageID uuid.UUID) (*ImageLineageInfo, error) {
	info := &ImageLineageInfo{
		ImageID: imageID,
	}

	// Count parents
	err := e.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM image_lineage WHERE image_id = $1", imageID).Scan(&info.ParentCount)
	if err != nil {
		return nil, fmt.Errorf("count parents: %w", err)
	}

	// Count direct children
	err = e.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM image_lineage WHERE parent_image_id = $1", imageID).Scan(&info.ChildCount)
	if err != nil {
		return nil, fmt.Errorf("count children: %w", err)
	}

	// Count all descendants
	descendants, err := e.GetAllDescendants(ctx, uuid.Nil, []uuid.UUID{imageID})
	if err != nil {
		return nil, fmt.Errorf("count descendants: %w", err)
	}
	info.TotalDescendants = len(descendants)

	// Find root ancestor
	path, err := e.GetLineagePath(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("get lineage path: %w", err)
	}

	if len(path) > 0 {
		root := path[0].ImageID
		if root != imageID {
			info.RootAncestor = &root
		}
		info.MaxDepth = path[len(path)-1].LineageDepth
	}

	return info, nil
}
