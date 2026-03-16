package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// listSkills 获取所有技能列表
func (h *Handler) listSkills(c *gin.Context) {
	skills, err := h.skillService.ListSkills()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Items: skills,
		Total: int64(len(skills)),
	})
}

// getSkill 获取单个技能详情
func (h *Handler) getSkill(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "skill name is required"})
		return
	}

	skill, err := h.skillService.GetSkill(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if skill == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "skill not found"})
		return
	}

	c.JSON(http.StatusOK, skill)
}
